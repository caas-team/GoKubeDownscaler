import React, { useState, useEffect } from "react";
import Heading from "@theme/Heading";
import useBaseUrl from "@docusaurus/useBaseUrl";
import { useColorMode } from "@docusaurus/theme-common";
import styles from "./styles.module.css";

/** Returns true only on phone-sized screens (< 640 px — Tailwind `sm` breakpoint). */
function useIsPhone(): boolean {
  const [isPhone, setIsPhone] = useState(false);
  useEffect(() => {
    const mq = window.matchMedia("(max-width: 639px)");
    setIsPhone(mq.matches);
    const handler = (e: MediaQueryListEvent) => setIsPhone(e.matches);
    mq.addEventListener("change", handler);
    return () => mq.removeEventListener("change", handler);
  }, []);
  return isPhone;
}

const TIMEZONES = [
  "America/Los_Angeles",
  "Asia/Tokyo",
  "Australia/Brisbane",
  "Europe/Bucharest",
  "America/Santiago",
  "Europe/Berlin",
];

const TYPING_SPEED_MS = 80;
const DELETING_SPEED_MS = 40;
const CYCLE_INTERVAL_MS = 8000;

function useTypingCycle(items: string[], disabled = false): string {
  const [index, setIndex] = useState(0);
  const [displayed, setDisplayed] = useState(items[0]);
  const [phase, setPhase] = useState<"typing" | "waiting" | "deleting">("waiting");

  useEffect(() => {
    if (disabled) return;
    const current = items[index];

    if (phase === "waiting") {
      const t = setTimeout(() => setPhase("deleting"), CYCLE_INTERVAL_MS);
      return () => clearTimeout(t);
    }

    if (phase === "deleting") {
      if (displayed.length === 0) {
        setIndex((i) => (i + 1) % items.length);
        setPhase("typing");
        return;
      }
      const t = setTimeout(
        () => setDisplayed((d) => d.slice(0, -1)),
        DELETING_SPEED_MS,
      );
      return () => clearTimeout(t);
    }

    if (phase === "typing") {
      if (displayed.length === current.length) {
        setPhase("waiting");
        return;
      }
      const t = setTimeout(
        () => setDisplayed(current.slice(0, displayed.length + 1)),
        TYPING_SPEED_MS,
      );
      return () => clearTimeout(t);
    }
  }, [phase, displayed, index, items]);

  return displayed;
}

function Terminal() {
  const isPhone = useIsPhone();
  const timezone = useTypingCycle(TIMEZONES, isPhone);
  const displayedTimezone = isPhone ? "Asia/Tokyo" : timezone;
  return (
    <div className={styles.terminal}>
      {/* Title bar */}
      <div className={styles.titleBar}>
        <span className={styles.dot} style={{ background: "#ff5f57" }} />
        <span className={styles.dot} style={{ background: "#febc2e" }} />
        <span className={styles.dot} style={{ background: "#28c840" }} />
        <span className={styles.fileName}>configmap.yaml</span>
      </div>
      {/* Code body */}
      <pre className={styles.code}>
        <code>
          <span className={styles.keyword}>apiVersion</span>
          <span className={styles.punct}>: </span>
          <span className={styles.value}>v1</span>
          {"\n"}
          <span className={styles.keyword}>kind</span>
          <span className={styles.punct}>: </span>
          <span className={styles.value}>ConfigMap</span>
          {"\n"}
          <span className={styles.keyword}>metadata</span>
          <span className={styles.punct}>:</span>
          {"\n"}
          {"  "}
          <span className={styles.keyword}>name</span>
          <span className={styles.punct}>: </span>
          <span className={styles.value}>kube-downscaler</span>
          {"\n"}
          {"  "}
          <span className={styles.keyword}>namespace</span>
          <span className={styles.punct}>: </span>
          <span className={styles.value}>kube-downscaler</span>
          {"\n"}
          <span className={styles.keyword}>data</span>
          <span className={styles.punct}>:</span>
          {"\n"}
          {"  "}
          <span className={styles.annotation}>DEFAULT_UPTIME</span>
          <span className={styles.punct}>: </span>
          <span className={styles.scheduleValue}>
            Mon-Fri 09:00-17:00 {displayedTimezone}
            {!isPhone && <span className={styles.cursor}>|</span>}
          </span>
          {"\n"}
          {"  "}
          <span className={styles.annotation}>EXCLUDE_NAMESPACES</span>
          <span className={styles.punct}>: </span>
          <span className={styles.value}>kube-system,cilium,kube-downscaler</span>
        </code>
      </pre>
    </div>
  );
}

export default function HowItWorks(): JSX.Element {
  const { colorMode } = useColorMode();
  const darkSrc  = useBaseUrl("/img/how-it-works-dark.png");
  const lightSrc = useBaseUrl("/img/how-it-works-light.png");
  const previewSrc = colorMode === "dark" ? darkSrc : lightSrc;
  return (
    <section className={styles.section}>
      <div className={styles.inner}>
        {/* Heading + description — always visible */}
        <div className={`${styles.textBlock} animate-fade-down animate-once animate-delay-0`}>
          <Heading as="h2" className={styles.headline}>
            How It Works
          </Heading>
          <p className={styles.body}>
            The most common way to use GoKubeDownscaler is to set a global
            schedule. The controller continuously read the
            desired configuration and scales down workloads when needed. Once the
            downscaling window ends, the controller brings the workload back
            at its original state.
          </p>
        </div>

        {/* Terminal — always visible */}
        <div className={`${styles.terminalBlock} animate-fade-down animate-once animate-delay-300`}>
          <Terminal />
        </div>

        {/* Image — hidden on small screens or when image is missing */}
        <div className={`${styles.visualBlock} animate-fade-down animate-once animate-delay-500`}>
          <img
            src={previewSrc}
            alt="GoKubeDownscaler scheduling dashboard showing workloads being scaled down during off-hours"
            className={styles.previewImage}
            width={1200}
            height={675}
            loading="lazy"
            decoding="async"
            onError={(e) => {
              (e.currentTarget.parentElement as HTMLElement).style.display = "none";
            }}
          />
        </div>
      </div>
    </section>
  );
}
