import { useEffect, useRef } from "react";
import { useNavigate } from "react-router-dom";

import { Counter, RsIcon as Icon, Tape } from "@/shared/ui/realsync";

const STREAM = [
  { t: "14:22:41", lvl: "ok", src: "session.finished", v: "backend · score 91" },
  { t: "14:18:02", lvl: "ok", src: "resume.parsed", v: "cv_v4.pdf · 7.4s · 92% conf" },
  { t: "14:12:55", lvl: "warn", src: "rate.limit", v: "ws reconnect · attempt 2" },
  { t: "14:08:17", lvl: "ok", src: "auth.refresh", v: "token rotated · ok" },
  { t: "13:54:09", lvl: "ok", src: "model.load", v: "haiku-4.5 · 218ms" },
];

const useParallax = () => {
  const ref = useRef<HTMLElement | null>(null);
  useEffect(() => {
    const onScroll = () => {
      if (!ref.current) return;
      const y = window.scrollY;
      ref.current.querySelectorAll<HTMLElement>("[data-px]").forEach((el) => {
        const speed = Number(el.dataset.px || 0);
        el.style.transform = `translate3d(0, ${y * speed}px, 0)`;
      });
    };
    window.addEventListener("scroll", onScroll, { passive: true });
    return () => window.removeEventListener("scroll", onScroll);
  }, []);
  return ref;
};

const useMagnetic = (strength = 0.25) => {
  const ref = useRef<HTMLSpanElement | null>(null);
  useEffect(() => {
    const el = ref.current;
    if (!el) return;
    const onMove = (e: MouseEvent) => {
      const r = el.getBoundingClientRect();
      const dx = (e.clientX - (r.left + r.width / 2)) * strength;
      const dy = (e.clientY - (r.top + r.height / 2)) * strength;
      el.style.transform = `translate(${dx}px, ${dy}px)`;
    };
    const onLeave = () => {
      el.style.transform = "";
    };
    el.addEventListener("mousemove", onMove);
    el.addEventListener("mouseleave", onLeave);
    return () => {
      el.removeEventListener("mousemove", onMove);
      el.removeEventListener("mouseleave", onLeave);
    };
  }, [strength]);
  return ref;
};

export default function HomePage() {
  const navigate = useNavigate();
  const heroRef = useParallax();
  const ctaRef = useMagnetic(0.2);
  const ctaRef2 = useMagnetic(0.2);

  return (
    <main className="page" data-screen-label="01 Home" ref={heroRef as React.RefObject<HTMLElement>}>
      <div className="sysbar reveal" style={{ marginBottom: 28 }}>
        <span><span className="dot"></span><span className="k">status</span><span className="v">operational</span></span>
        <span><span className="k">build</span><span className="v">v2.4.18-f7a2c3</span></span>
        <span><span className="k">latency p99</span><span className="v">218ms</span></span>
        <span><span className="k">uptime</span><span className="v">99.97%</span></span>
        <span><span className="k">region</span><span className="v">eu-central</span></span>
      </div>

      <section className="home-hero grid-bg" style={{ position: "relative", overflow: "hidden" }}>
        <span className="crosshair" style={{ top: 0, left: -6 }}></span>
        <span className="crosshair" style={{ top: 0, right: -6 }}></span>

        <span
          className="giant-letter"
          data-px="-0.08"
          style={{ fontSize: "clamp(360px, 44vw, 620px)", right: "-3vw", top: "-8vw", zIndex: 0 }}
          aria-hidden="true"
        >Я</span>
        <span
          className="giant-letter fill"
          data-px="-0.04"
          style={{ fontSize: "clamp(360px, 44vw, 620px)", right: "-2.6vw", top: "-7.7vw", zIndex: 0 }}
          aria-hidden="true"
        >Я</span>

        <div style={{ position: "relative", zIndex: 2 }}>
          <span className="eyebrow">Платформа RealSync · v2.4 · 2026</span>
          <h1 className="expr-headline" style={{ fontSize: "clamp(48px, 6.6vw, 104px)", marginTop: 16 }}>
            <span className="bold">Технология</span> <span className="ital">интервью,</span><br />
            которая <span className="ital underline">работает</span><br />
            <span className="light">просто и понятно</span><span className="cursor-blink"></span>
          </h1>
          <p className="lede">
            Исследуйте весь продукт в режиме фронтенда с богатыми мок-данными. AI-симулятор, разбор резюме и карьерный трек — в одном рабочем пространстве, без шумных дашбордов и градиентов.
          </p>
          <div className="home-hero-actions">
            <span className="magnetic" ref={ctaRef}>
              <button className="btn btn--primary" onClick={() => navigate("/interview")} type="button">
                Начать интервью <Icon name="arrow" />
              </button>
            </span>
            <span className="magnetic" ref={ctaRef2}>
              <button className="btn btn--ghost" onClick={() => navigate("/career-center")} type="button">
                Карьерный центр
              </button>
            </span>
          </div>
          <div className="wire" style={{ marginTop: 24 }}>open source · MIT · build {new Date().toISOString().slice(0, 10)}</div>
        </div>

        <aside className="home-hero-side" style={{ position: "relative", zIndex: 2 }}>
          <span className="eyebrow">Сегодня в системе</span>
          <div className="home-stat">
            <div>
              <div className="home-stat-label">Активных интервью</div>
              <div className="home-stat-value mono"><Counter target={1284} /></div>
            </div>
            <div className="home-stat-suffix">сессий</div>
          </div>
          <div className="home-stat">
            <div>
              <div className="home-stat-label">Среднее время отклика</div>
              <div className="home-stat-value mono"><Counter target={0.4} decimals={1} suffix="s" /></div>
            </div>
            <div className="home-stat-suffix">p50</div>
          </div>
          <div className="home-stat">
            <div>
              <div className="home-stat-label">Точность оценки</div>
              <div className="home-stat-value mono"><Counter target={94} suffix="%" /></div>
            </div>
            <div className="home-stat-suffix">vs human</div>
          </div>
        </aside>
      </section>

      <section className="grid-2" style={{ padding: "64px 0", borderBottom: "1px solid var(--line)", gap: 28 }}>
        <div className="brutal">
          <h4>// product.spec.v2</h4>
          <div className="row"><span>name</span><span>realsync-core</span></div>
          <div className="row"><span>kind</span><span>interview.platform</span></div>
          <div className="row"><span>language</span><span>go + react + py</span></div>
          <div className="row"><span>license</span><span>mit</span></div>
          <div className="row"><span>models</span><span>haiku-4.5 / sonnet-4.5</span></div>
          <div className="row"><span>regions</span><span>eu-central · us-east</span></div>
          <div style={{ marginTop: 16, fontFamily: "var(--f-mono)", fontSize: 11, color: "var(--muted)" }}>
            // built for engineers who interview engineers.
          </div>
        </div>

        <div>
          <div className="row-between" style={{ alignItems: "baseline", marginBottom: 14 }}>
            <span className="eyebrow">Live event stream</span>
            <span className="mono" style={{ fontSize: 11, color: "var(--muted)" }}>last 30 min · 1284 events</span>
          </div>
          <div className="datastream">
            {STREAM.map((r, i) => (
              <div className="datastream-row" key={i} style={{ animationDelay: `${i * 80}ms` }}>
                <span className="t">{r.t}</span>
                <span className={`lvl ${r.lvl}`}></span>
                <span><span className="src">{r.src}</span> <span className="v">{r.v}</span></span>
                <span className="mono" style={{ color: "var(--muted-2)", fontSize: 10 }}>→</span>
              </div>
            ))}
          </div>
        </div>
      </section>

      <section className="home-features">
        {[
          { n: "01", t: "AI-симуляция интервью", d: "Живой диалог с моделью, которая адаптируется под уровень и роль. Theory и practice треки, code-workspace внутри сессии." },
          { n: "02", t: "Резюме и профиль рядом", d: "Загрузите PDF — получите карту навыков, рекомендованные треки и risks/strengths. Всё в одном рабочем пространстве." },
          { n: "03", t: "Чёткие рекомендации", d: "Не «вы молодец», а конкретный план: что повторить, какую слабую тему добрать, какой следующий мок запустить." },
        ].map((f, i) => (
          <div className={`home-feature reveal reveal-${i + 1}`} key={f.n}>
            <div className="home-feature-num">{f.n} / 03</div>
            <h3 className="wonk">
              {f.t.split("").map((ch, idx) => <span key={idx}>{ch === " " ? " " : ch}</span>)}
            </h3>
            <p>{f.d}</p>
            <div className="arrow">подробнее <span>→</span></div>
          </div>
        ))}
      </section>

      <Tape items={[
        "28 интервью пройдено",
        "Backend · Go · 91%",
        "Streak 6 дней",
        "Adaptive challenge активен",
        "Резюме обновлено",
        "Frontend · React · в очереди",
        "Public profile · v3",
      ]} />

      <section className="home-cta">
        <h2 className="expr-headline" style={{ fontSize: "clamp(40px, 5vw, 72px)" }}>
          Готовы продолжить?<br />
          <span className="ital">Откройте свой</span> <span className="pill">dashboard</span>.
        </h2>
        <button className="btn btn--accent" onClick={() => navigate("/dashboard")} type="button">
          Перейти на панель <Icon name="arrow" />
        </button>
      </section>
    </main>
  );
}
