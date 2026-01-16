import React from 'react';
import { AgentCard } from './AgentCard';
import type { Agent } from '../../../types';

export interface AgentGridProps {
  agents: Agent[];
  onSync: (id: string) => void;
  onViewDetails: (agent: Agent) => void;
  onEdit: (agent: Agent) => void;
  onDelete: (id: string, name: string) => void;
}

export const AgentGrid: React.FC<AgentGridProps> = ({
  agents,
  onSync,
  onViewDetails,
  onEdit,
  onDelete,
}) => {
  return (
    <div className="col-12">
      <div className="row row-deck row-cards">
        {agents.map((agent) => (
          <AgentCard
            key={agent.id}
            agent={agent}
            onSync={onSync}
            onViewDetails={onViewDetails}
            onEdit={onEdit}
            onDelete={onDelete}
          />
        ))}
      </div>
    </div>
  );
};
