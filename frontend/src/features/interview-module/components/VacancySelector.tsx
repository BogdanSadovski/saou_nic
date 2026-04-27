import { useMemo } from "react";

import type { VacancyOption } from "../types";

type Props = {
  options: VacancyOption[];
  query: string;
  selectedId: string;
  onQueryChange: (value: string) => void;
  onSelect: (vacancy: VacancyOption) => void;
};

const normalize = (value: string) => value.toLowerCase().trim();

export function VacancySelector({ options, query, selectedId, onQueryChange, onSelect }: Props) {
  const filtered = useMemo(() => {
    const needle = normalize(query);
    if (!needle) {
      return options;
    }

    return options.filter((vacancy) => {
      const haystack = [
        vacancy.title,
        vacancy.category,
        vacancy.description,
        ...vacancy.searchTerms,
        ...vacancy.primarySkills,
      ]
        .join(" ")
        .toLowerCase();

      return haystack.includes(needle);
    });
  }, [options, query]);

  return (
    <div className="interview-field vacancy-field">
      <label htmlFor="vacancy-search">Вакансия</label>
      <input
        id="vacancy-search"
        className="vacancy-search-input"
        value={query}
        onChange={(event) => onQueryChange(event.target.value)}
        placeholder="Поиск: backend, mobile, security, manager..."
      />
      <div className="vacancy-list" role="listbox" aria-label="Список вакансий">
        {filtered.map((vacancy) => {
          const selected = vacancy.id === selectedId;
          return (
            <button
              key={vacancy.id}
              type="button"
              className={selected ? "vacancy-item selected" : "vacancy-item"}
              onClick={() => onSelect(vacancy)}
            >
              <span className="vacancy-title">{vacancy.title}</span>
              <span className="vacancy-category">{vacancy.category}</span>
              <span className="vacancy-description">{vacancy.description}</span>
            </button>
          );
        })}
      </div>
    </div>
  );
}
