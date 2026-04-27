import type { InterviewRole } from "../types";

type Props = {
  value: InterviewRole;
  onChange: (value: InterviewRole) => void;
};

const roles: InterviewRole[] = ["Frontend", "Backend", "DevOps", "ML", "Mobile", "Data"];

export function RoleSelector({ value, onChange }: Props) {
  return (
    <div className="interview-field">
      <label htmlFor="role">Роль</label>
      <select id="role" value={value} onChange={(event) => onChange(event.target.value as InterviewRole)}>
        {roles.map((role) => (
          <option key={role} value={role}>
            {role}
          </option>
        ))}
      </select>
    </div>
  );
}
