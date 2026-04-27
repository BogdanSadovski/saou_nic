import { AuthForm } from "@/features/auth/AuthForm";
import { GithubConnectCard } from "@/features/github-connect/GithubConnectCard";

export default function AuthPage() {
  return (
    <section className="page auth-page two-col">
      <AuthForm />
      <GithubConnectCard />
    </section>
  );
}
