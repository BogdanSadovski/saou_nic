import React from 'react';

interface Interview {
  id: string;
  candidate: string;
  role: string;
  date: string;
  score: number;
  status: 'completed' | 'scheduled' | 'in-progress';
}

interface RecentInterviewsProps {
  interviews?: Interview[];
  onView?: (id: string) => void;
  limit?: number;
}

const defaultInterviews: Interview[] = [
  {
    id: '1',
    candidate: 'Alice Smith',
    role: 'Frontend Developer',
    date: '2026-04-05',
    score: 85,
    status: 'completed',
  },
  {
    id: '2',
    candidate: 'Bob Johnson',
    role: 'Backend Developer',
    date: '2026-04-06',
    score: 72,
    status: 'completed',
  },
  {
    id: '3',
    candidate: 'Carol Williams',
    role: 'Full Stack Developer',
    date: '2026-04-07',
    score: 0,
    status: 'scheduled',
  },
];

const RecentInterviews: React.FC<RecentInterviewsProps> = ({
  interviews = defaultInterviews,
  onView,
  limit = 5,
}) => {
  const displayInterviews = interviews.slice(0, limit);

  return (
    <div className="recent-interviews">
      <h3 className="recent-interviews__title">Последние интервью</h3>
      <table className="recent-interviews__table">
        <thead>
          <tr>
            <th>Кандидат</th>
            <th>Роль</th>
            <th>Дата</th>
            <th>Баллы</th>
            <th>Статус</th>
            <th>Действие</th>
          </tr>
        </thead>
        <tbody>
          {displayInterviews.map((interview) => (
            <tr key={interview.id}>
              <td>{interview.candidate}</td>
              <td>{interview.role}</td>
              <td>{interview.date}</td>
              <td>{interview.score > 0 ? `${interview.score}%` : '—'}</td>
              <td>
                <span className={`recent-interviews__status recent-interviews__status--${interview.status}`}>
                  {interview.status}
                </span>
              </td>
              <td>
                <button
                  className="recent-interviews__view-btn"
                  onClick={() => onView?.(interview.id)}
                >
                  Открыть
                </button>
              </td>
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  );
};

export default RecentInterviews;
