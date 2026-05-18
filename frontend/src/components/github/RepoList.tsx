import React from 'react';

interface Repository {
  id: number;
  name: string;
  description?: string;
  language?: string;
  stars: number;
  forks: number;
  updatedAt: string;
}

interface RepoListProps {
  repositories?: Repository[];
  onSelect?: (repo: Repository) => void;
  isLoading?: boolean;
}

const defaultRepos: Repository[] = [
  {
    id: 1,
    name: 'my-project',
    description: 'A sample project',
    language: 'TypeScript',
    stars: 42,
    forks: 5,
    updatedAt: '2026-04-01',
  },
  {
    id: 2,
    name: 'api-service',
    description: 'Backend API service',
    language: 'Python',
    stars: 18,
    forks: 3,
    updatedAt: '2026-03-28',
  },
];

const RepoList: React.FC<RepoListProps> = ({
  repositories = defaultRepos,
  onSelect,
  isLoading = false,
}) => {
  if (isLoading) {
    return <div className="repo-list repo-list--loading">Загрузка репозиториев...</div>;
  }

  return (
    <div className="repo-list">
      <h3 className="repo-list__title">Репозитории</h3>
      <ul className="repo-list__list">
        {repositories.map((repo) => (
          <li
            key={repo.id}
            className="repo-list__item"
            onClick={() => onSelect?.(repo)}
            role="button"
            tabIndex={0}
          >
            <div className="repo-list__item-header">
              <span className="repo-list__item-name">{repo.name}</span>
              {repo.language && (
                <span className="repo-list__item-language">{repo.language}</span>
              )}
            </div>
            {repo.description && (
              <p className="repo-list__item-description">{repo.description}</p>
            )}
            <div className="repo-list__item-stats">
              <span className="repo-list__stat">
                {'\u2B50'} {repo.stars}
              </span>
              <span className="repo-list__stat">
                {'\u{1F500}'} {repo.forks}
              </span>
              <span className="repo-list__stat">
                Обновлён {repo.updatedAt}
              </span>
            </div>
          </li>
        ))}
      </ul>
    </div>
  );
};

export default RepoList;
