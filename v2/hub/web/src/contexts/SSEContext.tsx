import React, { createContext, useContext, useEffect, useState, ReactNode } from 'react';
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
  const [eventSource, setEventSource] = useState<EventSource | null>(null);
  const [subscribers, setSubscribers] = useState<Map<string, Set<(event: SSEEvent) => void>>>(new Map());
  const { token } = useAuth();

  useEffect(() => {
    if (!token) {
      // Close existing connection if token is not available
      if (eventSource) {
        eventSource.close();
        setEventSource(null);
        setConnected(false);
      }
      return;
    }

    // Create SSE connection
    const baseUrl = import.meta.env.VITE_API_URL || '';
    const es = new EventSource(`${baseUrl}/events`, {
      withCredentials: true,
    });

    es.onopen = () => {
      console.log('SSE connection established');
      setConnected(true);
    };

    es.onerror = (error) => {
      console.error('SSE connection error:', error);
      setConnected(false);
      
      // Reconnect after 5 seconds
      setTimeout(() => {
        if (es.readyState === EventSource.CLOSED) {
          console.log('Attempting to reconnect SSE...');
          // Recursively call this effect by updating a dependency
        }
      }, 5000);
    };

    // Generic message handler
    es.onmessage = (event) => {
      try {
        const data = JSON.parse(event.data);
        handleEvent('message', data);
      } catch (error) {
        console.error('Failed to parse SSE message:', error);
      }
    };

    // Specific event handlers
    const eventTypes = [
      'agent.status.update',
      'agent.registered',
      'agent.heartbeat',
      'execution.status.update',
      'execution.log.update',
      'task.created',
      'task.updated',
      'task.deleted',
    ];

    eventTypes.forEach(eventType => {
      es.addEventListener(eventType, (event: MessageEvent) => {
        try {
          const data = JSON.parse(event.data);
          handleEvent(eventType, data);
        } catch (error) {
          console.error(`Failed to parse ${eventType} event:`, error);
        }
      });
    });

    const handleEvent = (type: string, data: any) => {
      const newEvent: SSEEvent = {
        id: `${Date.now()}-${Math.random()}`,
        type,
        data,
        timestamp: new Date(),
      };

      setEvents(prev => [...prev.slice(-99), newEvent]); // Keep last 100 events
      setLastEvent(newEvent);

      // Notify subscribers
      const typeSubscribers = subscribers.get(type);
      if (typeSubscribers) {
        typeSubscribers.forEach(callback => callback(newEvent));
      }

      // Notify wildcard subscribers
      const wildcardSubscribers = subscribers.get('*');
      if (wildcardSubscribers) {
        wildcardSubscribers.forEach(callback => callback(newEvent));
      }
    };

    setEventSource(es);

    // Cleanup
    return () => {
      es.close();
      setConnected(false);
    };
  }, [token]); // Reconnect when token changes

  const clearEvents = () => {
    setEvents([]);
    setLastEvent(null);
  };

  const subscribe = (eventType: string, callback: (event: SSEEvent) => void) => {
    setSubscribers(prev => {
      const newMap = new Map(prev);
      if (!newMap.has(eventType)) {
        newMap.set(eventType, new Set());
      }
      newMap.get(eventType)!.add(callback);
      return newMap;
    });

    // Return unsubscribe function
    return () => {
      setSubscribers(prev => {
        const newMap = new Map(prev);
        const typeSubscribers = newMap.get(eventType);
        if (typeSubscribers) {
          typeSubscribers.delete(callback);
          if (typeSubscribers.size === 0) {
            newMap.delete(eventType);
          }
        }
        return newMap;
      });
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