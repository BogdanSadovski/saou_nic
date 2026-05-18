import React from 'react';

interface Candidate {
  id: string;
  name: string;
  email: string;
  appliedRole: string;
  status: 'new' | 'interviewed' | 'hired' | 'rejected';
  avatarUrl?: string;
}

interface CandidateListProps {
  candidates?: Candidate[];
  onSelect?: (id: string) => void;
  filter?: Candidate['status'];
}

const defaultCandidates: Candidate[] = [
  { id: '1', name: 'Alice Smith', email: 'alice@example.com', appliedRole: 'Frontend Dev', status: 'interviewed' },
  { id: '2', name: 'Bob Johnson', email: 'bob@example.com', appliedRole: 'Backend Dev', status: 'new' },
  { id: '3', name: 'Carol Williams', email: 'carol@example.com', appliedRole: 'Full Stack', status: 'hired' },
];

const CandidateList: React.FC<CandidateListProps> = ({
  candidates = defaultCandidates,
  onSelect,
  filter,
}) => {
  const filtered = filter
    ? candidates.filter((c) => c.status === filter)
    : candidates;

  return (
    <div className="candidate-list">
      <h3 className="candidate-list__title">Кандидаты</h3>
      <ul className="candidate-list__list">
        {filtered.map((candidate) => (
          <li
            key={candidate.id}
            className="candidate-list__item"
            onClick={() => onSelect?.(candidate.id)}
            role="button"
            tabIndex={0}
          >
            <div className="candidate-list__avatar">
              {candidate.avatarUrl ? (
                <img src={candidate.avatarUrl} alt={candidate.name} />
              ) : (
                <span className="candidate-list__avatar-placeholder">
                  {candidate.name.charAt(0)}
                </span>
              )}
            </div>
            <div className="candidate-list__info">
              <span className="candidate-list__name">{candidate.name}</span>
              <span className="candidate-list__role">{candidate.appliedRole}</span>
            </div>
            <span className={`candidate-list__status candidate-list__status--${candidate.status}`}>
              {candidate.status}
            </span>
          </li>
        ))}
      </ul>
    </div>
  );
};

export default CandidateList;
