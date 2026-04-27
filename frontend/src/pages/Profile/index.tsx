import { useState } from "react";

import { useUserStore } from "@/app/store";
import { useTranslation } from "@/shared/i18n";
import { GithubConnectCard } from "@/features/github-connect/GithubConnectCard";
import { FloatingInput, GlassButton, GlassCard, useToast } from "@/shared/ui";

export default function ProfilePage() {
  const user = useUserStore((state) => state.user);
  const updateProfile = useUserStore((state) => state.updateProfile);
  const { pushToast } = useToast();
  const t = useTranslation();

  const [fullName, setFullName] = useState(user.fullName);
  const [email, setEmail] = useState(user.email);

  return (
    <section className="page two-col">
      <GlassCard>
        <h1>{t.profileTitle}</h1>
        <p className="muted">{t.manageLocalIdentity}</p>

        <FloatingInput
          label={t.fullName}
          onChange={(event) => setFullName(event.target.value)}
          value={fullName}
        />
        <FloatingInput
          label={t.email}
          onChange={(event) => setEmail(event.target.value)}
          value={email}
        />

        <p className="muted profile-role">{t.role}: {user.role}</p>

        <GlassButton
          onClick={async () => {
            try {
              await updateProfile({ fullName, email });
              pushToast(t.profileUpdated);
            } catch {
              pushToast("Не удалось обновить профиль");
            }
          }}
          type="button"
        >
          {t.saveChanges}
        </GlassButton>
      </GlassCard>
      <GithubConnectCard />
    </section>
  );
}
