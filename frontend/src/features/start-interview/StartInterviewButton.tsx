import { useNavigate } from "react-router-dom";

import { useTranslation } from "@/shared/i18n";
import { GlassButton } from "@/shared/ui";

export function StartInterviewButton() {
  const navigate = useNavigate();
  const t = useTranslation();

  return (
    <GlassButton onClick={() => navigate("/interview")}>{t.startInterview}</GlassButton>
  );
}
