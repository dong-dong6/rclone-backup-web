import React, { createContext, useContext, useEffect, useState, useRef, ReactNode } from 'react';
import { useAuth } from './AuthContext';

interface SSEEvent {
  id: string;
  type: string;
  data: any;
  timestamp: Date;
}

interface SSEContextType {
  events: SSEEvent[];
  connected: boolean;
  lastEvent: SSEEvent | null;
  clearEvents: () => void;
  subscribe: (eventType: string, callback: (event: SSEEvent) => void) => () => void;
}

const SSEContext = createContext<SSEContextType | undefined>(undefined);

export const useSSE = () => {
  const context = useContext(SSEContext);
  if (!context) {
    throw new Error('useSSE must be used within an SSEProvider');
  }
  return context;
};

interface SSEProviderProps {
  children: ReactNode;
}

export const SSEProvider: React.FC<SSEProviderProps> = ({ children }) => {
  const [events, setEvents] = useState<SSEEvent[]>([]);
  const [connected, setConnected] = useState(false);
  const [lastEvent, setLastEvent] = useState<SSEEvent | null>(null);
  const subscribersRef = useRef<Map<string, Set<(event: SSEEvent) => void>>>(new Map());
  const { token } = useAuth();
  const eventSourceRef = useRef<EventSource | null>(null);
  const reconnectTimerRef = useRef<number | null>(null);
  const retryAttemptRef = useRef(0);

  useEffect(() => {
    const cleanup = () => {
      setConnected(false);
      if (reconnectTimerRef.current !== null) {
        window.clearTimeout(reconnectTimerRef.current);
        reconnectTimerRef.current = null;
      }
      if (eventSourceRef.current) {
        eventSourceRef.current.close();
        eventSourceRef.current = null;
      }
    };

    if (!token) {
      cleanup();
      return;
    }

    const baseUrl = import.meta.env.VITE_API_URL || '';

    const handleEvent = (type: string, data: any) => {
      const newEvent: SSEEvent = {
        id: `${Date.now()}-${Math.random()}`,
        type,
        data,
        timestamp: new Date(),
      };

      setEvents(prev => [...prev.slice(-99), newEvent]); // Keep last 100 events
      setLastEvent(newEvent);

      // Notify subscribers (using ref to avoid stale closure)
      const typeSubscribers = subscribersRef.current.get(type);
      if (typeSubscribers) {
        typeSubscribers.forEach(callback => callback(newEvent));
      }

      // Notify wildcard subscribers
      const wildcardSubscribers = subscribersRef.current.get('*');
      if (wildcardSubscribers) {
        wildcardSubscribers.forEach(callback => callback(newEvent));
      }
    };

    const eventTypes = [
      'agent.status.update',
      'agent.registered',
      'agent.heartbeat',
      'execution.status.update',
      'execution.log.update',
      'task.dispatched',
      'task.created',
      'task.updated',
      'task.deleted',
    ];

    const connect = () => {
      if (!token) return;

      if (eventSourceRef.current) {
        eventSourceRef.current.close();
        eventSourceRef.current = null;
      }

      const es = new EventSource(`${baseUrl}/events?token=${encodeURIComponent(token)}`, {
        withCredentials: true,
      });
      eventSourceRef.current = es;

      es.onopen = () => {
        retryAttemptRef.current = 0;
        setConnected(true);
      };

      es.onerror = () => {
        setConnected(false);
        es.close();

        if (reconnectTimerRef.current !== null) return;

        retryAttemptRef.current += 1;
        const backoffMs = Math.min(30_000, 1_000 * Math.pow(2, retryAttemptRef.current - 1));
        const jitterMs = Math.floor(Math.random() * 250);
        reconnectTimerRef.current = window.setTimeout(() => {
          reconnectTimerRef.current = null;
          connect();
        }, backoffMs + jitterMs);
      };

      es.onmessage = (event) => {
        try {
          const data = JSON.parse(event.data);
          handleEvent('message', data);
        } catch (error) {
          console.error('Failed to parse SSE message:', error);
        }
      };

      eventTypes.forEach((eventType) => {
        es.addEventListener(eventType, (event: MessageEvent) => {
          try {
            const data = JSON.parse(event.data);
            handleEvent(eventType, data);
          } catch (error) {
            console.error(`Failed to parse ${eventType} event:`, error);
          }
        });
      });
    };

    connect();

    // Cleanup
    return () => {
      cleanup();
    };
  }, [token]); // Reconnect when token changes

  const clearEvents = () => {
    setEvents([]);
    setLastEvent(null);
  };

  const subscribe = (eventType: string, callback: (event: SSEEvent) => void) => {
    // Directly modify ref to avoid stale closure issues
    if (!subscribersRef.current.has(eventType)) {
      subscribersRef.current.set(eventType, new Set());
    }
    subscribersRef.current.get(eventType)!.add(callback);

    // Return unsubscribe function
    return () => {
      const typeSubscribers = subscribersRef.current.get(eventType);
      if (typeSubscribers) {
        typeSubscribers.delete(callback);
        if (typeSubscribers.size === 0) {
          subscribersRef.current.delete(eventType);
        }
      }
    };
  };

  const value = {
    events,
    connected,
    lastEvent,
    clearEvents,
    subscribe,
  };

  return <SSEContext.Provider value={value}>{children}</SSEContext.Provider>;
};
