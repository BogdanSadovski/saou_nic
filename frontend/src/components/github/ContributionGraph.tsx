import React from "react";

interface ContributionDay {
  date: string;
  count: number;
}

interface ContributionGraphProps {
  contributions?: ContributionDay[];
  username?: string;
  year?: number;
}

const ContributionGraph: React.FC<ContributionGraphProps> = ({
  contributions,
  username = "user",
  year = new Date().getFullYear(),
}) => {
  const defaultContributions: ContributionDay[] = Array.from({ length: 365 }, (_, i) => {
    const date = new Date(year, 0, 1);
    date.setDate(date.getDate() + i);
    return {
      date: date.toISOString().split("T")[0],
      count: 0,
    };
  });

  const data = contributions && contributions.length > 0 ? contributions : defaultContributions;
  const maxCount = Math.max(...data.map((d) => d.count), 1);

  const getIntensity = (count: number): string => {
    const ratio = count / maxCount;
    if (ratio === 0) return "contribution-graph__day--none";
    if (ratio < 0.25) return "contribution-graph__day--low";
    if (ratio < 0.5) return "contribution-graph__day--medium";
    if (ratio < 0.75) return "contribution-graph__day--high";
    return "contribution-graph__day--very-high";
  };

  return (
    <div className="contribution-graph">
      <h3 className="contribution-graph__title">
        Активность {username} ({year})
      </h3>
      <div className="contribution-graph__grid">
        {data.map((day) => (
          <div
            key={day.date}
            className={`contribution-graph__day ${getIntensity(day.count)}`}
            title={`${day.date}: ${day.count} коммитов`}
          />
        ))}
      </div>
      <div className="contribution-graph__legend">
        <span>Меньше</span>
        <div className="contribution-graph__day contribution-graph__day--none" />
        <div className="contribution-graph__day contribution-graph__day--low" />
        <div className="contribution-graph__day contribution-graph__day--medium" />
        <div className="contribution-graph__day contribution-graph__day--high" />
        <div className="contribution-graph__day contribution-graph__day--very-high" />
        <span>Больше</span>
      </div>
    </div>
  );
};

export default ContributionGraph;
