import React from 'react';

interface ResumeData {
  name: string;
  email: string;
  phone?: string;
  summary?: string;
  experience?: Array<{ title: string; company: string; period: string }>;
  education?: Array<{ degree: string; institution: string; year: string }>;
  skills?: string[];
}

interface ResumePreviewProps {
  resume?: ResumeData;
  variant?: 'compact' | 'detailed';
}

const defaultResume: ResumeData = {
  name: 'John Doe',
  email: 'john@example.com',
  summary: 'Experienced software engineer with a passion for building scalable applications.',
  experience: [
    { title: 'Senior Engineer', company: 'Tech Corp', period: '2020 - Present' },
  ],
  education: [{ degree: 'B.S. Computer Science', institution: 'University', year: '2020' }],
  skills: ['React', 'TypeScript', 'Node.js', 'Python'],
};

const ResumePreview: React.FC<ResumePreviewProps> = ({
  resume = defaultResume,
  variant = 'detailed',
}) => {
  return (
    <div className={`resume-preview resume-preview--${variant}`}>
      <header className="resume-preview__header">
        <h2 className="resume-preview__name">{resume.name}</h2>
        <p className="resume-preview__email">{resume.email}</p>
        {resume.phone && <p className="resume-preview__phone">{resume.phone}</p>}
      </header>

      {resume.summary && (
        <section className="resume-preview__section">
          <h3>Краткое описание</h3>
          <p>{resume.summary}</p>
        </section>
      )}

      {variant === 'detailed' && resume.experience && (
        <section className="resume-preview__section">
          <h3>Опыт работы</h3>
          {resume.experience.map((exp, idx) => (
            <div key={idx} className="resume-preview__experience">
              <h4>{exp.title}</h4>
              <p className="resume-preview__company">
                {exp.company} | {exp.period}
              </p>
            </div>
          ))}
        </section>
      )}

      {variant === 'detailed' && resume.education && (
        <section className="resume-preview__section">
          <h3>Образование</h3>
          {resume.education.map((edu, idx) => (
            <div key={idx} className="resume-preview__education">
              <h4>{edu.degree}</h4>
              <p>
                {edu.institution} | {edu.year}
              </p>
            </div>
          ))}
        </section>
      )}

      {resume.skills && (
        <section className="resume-preview__section">
          <h3>Навыки</h3>
          <div className="resume-preview__skills">
            {resume.skills.map((skill) => (
              <span key={skill} className="resume-preview__skill-tag">
                {skill}
              </span>
            ))}
          </div>
        </section>
      )}
    </div>
  );
};

export default ResumePreview;
