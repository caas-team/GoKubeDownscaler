import React, { useEffect, useRef, useState } from "react";
import Heading from "@theme/Heading";
import styles from "./styles.module.css";

/* ── Typing animation for Helm commands ── */

type CommandDef = { cmd: string; successMsg: string };

const HELM_COMMANDS: CommandDef[] = [
  {
    cmd: "helm repo add caas-team https://caas-team.github.io/helm-charts/",
    successMsg: "Repository added successfully",
  },
  {
    cmd: "helm install go-kube-downscaler caas-team/go-kube-downscaler",
    successMsg: "go-kube-downscaler installed",
  },
];

function InstallTerminal() {
  const [allDone] = useState(true);

  return (
    <div className={styles.terminal}>
      {/* Title bar — identical to HowItWorks */}
      <div className={styles.titleBar}>
        <span className={styles.dot} style={{ background: "#ff5f57" }} />
        <span className={styles.dot} style={{ background: "#febc2e" }} />
        <span className={styles.dot} style={{ background: "#28c840" }} />
        <span className={styles.fileName}>install.sh</span>
      </div>

      {/* Code body — same <pre><code> pattern as HowItWorks */}
      <pre className={styles.code}>
        <code>
          {/* All commands completed */}
          {HELM_COMMANDS.map((item, i) => (
            <React.Fragment key={i}>
              <span className={styles.prompt}>{"$ "}</span>
              <span className={styles.cmd}>{item.cmd}</span>
              {"\n"}
              <span className={styles.success}>{"✓ "}</span>
              <span className={styles.successMsg}>{item.successMsg}</span>
              {"\n\n"}
            </React.Fragment>
          ))}

          {/* Final messages */}
          {allDone && (
            <>
              <span className={styles.checkIcon}>{"✓ "}</span>
              <span className={styles.installed}>
                {"go-kube-downscaler installed"}
              </span>
              {"\n"}
              <span className={styles.rocket}>{"🚀 "}</span>
              <span className={styles.ready}>
                {"You are now ready to save 70% of your Kubernetes bill"}
              </span>
            </>
          )}
        </code>
      </pre>
    </div>
  );
}

/* ── Bar chart ── */

type BarDatum = {
  label: string;
  /** 0–100 percentage of the max bar height */
  pct: number;
  isAfter: boolean;
  savingsPct?: number; // Optional savings percentage for the savings line
};

const BARS: BarDatum[] = [
  { label: "Jan", pct: 93, isAfter: false, savingsPct: 0 },
  { label: "Feb", pct: 100, isAfter: false, savingsPct: 0 },
  { label: "Mar", pct: 96, isAfter: false, savingsPct: 0 },
  { label: "Apr", pct: 30, isAfter: true, savingsPct: 70 },
  { label: "May", pct: 28, isAfter: true, savingsPct: 70 },
  { label: "Jun", pct: 32, isAfter: true, savingsPct: 70 },
];

function CostChart() {
  const ref = useRef<HTMLDivElement>(null);
  const [visible, setVisible] = useState(false);
  const [isPhone, setIsPhone] = useState(false);

  useEffect(() => {
    const mq = window.matchMedia("(max-width: 899px)");
    setIsPhone(mq.matches);
    const handler = (e: MediaQueryListEvent) => setIsPhone(e.matches);
    mq.addEventListener("change", handler);
    return () => mq.removeEventListener("change", handler);
  }, []);

  useEffect(() => {
    if (isPhone) return; // Don't trigger animations on mobile
    const el = ref.current;
    if (!el) return;
    const obs = new IntersectionObserver(
      ([entry]) => {
        if (entry.isIntersecting) {
          setVisible(true);
          obs.disconnect();
        }
      },
      { threshold: 0.3 },
    );
    obs.observe(el);
    return () => obs.disconnect();
  }, [isPhone]);

  return (
    <div className={styles.chartWrapper} ref={ref}>
      <p className={styles.chartTitle}>Monthly Cloud Cost $</p>

      {/* Bars */}
      <div className={styles.barsRow}>
        {BARS.map((bar, i) => {
          const isInstallBar = bar.isAfter;
          return (
            <div key={bar.label} className={styles.barCol}>
              <div className={styles.barTrack}>
                <div
                  className={`${styles.barFill} ${isInstallBar ? styles.barAfter : styles.barBefore}`}
                  style={
                    {
                      "--bar-pct": `${bar.pct}%`,
                      animationDelay: isPhone ? "0ms" : `${i * 120}ms`,
                      animationPlayState:
                        isPhone || visible
                          ? isPhone
                            ? "none"
                            : "running"
                          : "paused",
                    } as React.CSSProperties
                  }
                />
              </div>
              <span className={styles.barLabel}>{bar.label}</span>
            </div>
          );
        })}
      </div>

      {/* Divider between before / after */}
      <div className={styles.installMarker}>
        <span className={styles.installMarkerLine} />
        <span className={styles.installMarkerText}>Install</span>
        <span className={styles.installMarkerLine} />
      </div>

      {/* Legend with 70% badge */}
      <div className={styles.legend}>
        <span className={styles.legendDot} data-variant="before" />
        <span className={styles.legendText}>Before</span>
        <span className={styles.legendDot} data-variant="after" />
        <span className={styles.legendText}>After</span>
        <span className={styles.savingBadgeInline}>
          −70% Of Kubernetes Costs
        </span>
      </div>

      {/* Calculation basis note */}
      <p
        style={{
          fontSize: "0.8rem",
          color: "var(--ifm-color-emphasis-600)",
          marginTop: "0.75rem",
          marginBottom: 0,
        }}
      >
        * This calculation is based on a 40-hour work week (scale-down during
        evenings, weekends)
      </p>
    </div>
  );
}

/* ── Section ── */

export default function InstallationShowcase(): JSX.Element {
  return (
    <section className={styles.section}>
      <div className={styles.inner}>
        {/* Heading */}
        <div
          className={`${styles.textBlock} animate-fade-down animate-once animate-delay-0`}
        >
          <Heading as="h2" className={styles.headline}>
            Install In 1 Minute, Save All Year Round
          </Heading>
          <p className={styles.body}>
            Install GoKubeDownscaler with Helm in under a minute and start
            saving on your cloud bill from day one.{" "}
            <strong>
              Teams using GoKubeDownscaler achieve 70% savings after adopting
              it. No code changes required.
            </strong>
          </p>
          {/* License text */}
          <p
            className="animate-fade-down text-sm opacity-70 mt-6"
            style={{ willChange: "transform", margin: "-0.5rem 0 0 0" }}
          >
            ✓ Free And Open Source
          </p>
        </div>

        {/* Two-column panel */}
        <div className={styles.panel}>
          {/* Left — terminal */}
          <div
            className={`${styles.panelLeft} animate-fade-down animate-once animate-delay-200`}
          >
            <InstallTerminal />
          </div>

          {/* Right — bar chart */}
          <div
            className={`${styles.panelRight} animate-fade-down animate-once animate-delay-400`}
          >
            <CostChart />
          </div>
        </div>
      </div>
    </section>
  );
}
