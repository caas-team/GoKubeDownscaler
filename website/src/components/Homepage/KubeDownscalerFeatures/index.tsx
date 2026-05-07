import React, { ReactNode, useState } from "react";
import clsx from "clsx";
import Heading from "@theme/Heading";
import { Button, GitHubButton } from "@site/src/components/Basic/Button";
import styles from "./styles.module.css";

/* ── Inline SVG icons ── */

function ClockSvg(props: React.ComponentProps<"svg">) {
  return (
    <svg {...props} focusable="false" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth={1.5} strokeLinecap="round" strokeLinejoin="round">
      <circle cx="12" cy="12" r="10" />
      <polyline points="12 6 12 12 16 14" />
    </svg>
  );
}

/*
function MoneySvg(props: React.ComponentProps<"svg">) {
  return (
    <svg {...props} focusable="false" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth={1.5} strokeLinecap="round" strokeLinejoin="round">
      <rect x="2" y="7" width="20" height="14" rx="2" ry="2" />
      <path d="M16 3H8a2 2 0 0 0-2 2v2h12V5a2 2 0 0 0-2-2z" />
      <line x1="12" y1="12" x2="12" y2="16" />
      <line x1="10" y1="14" x2="14" y2="14" />
    </svg>
  );
}

function LayersSvg(props: React.ComponentProps<"svg">) {
  return (
    <svg {...props} focusable="false" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth={1.5} strokeLinecap="round" strokeLinejoin="round">
      <polygon points="12 2 2 7 12 12 22 7 12 2" />
      <polyline points="2 17 12 22 22 17" />
      <polyline points="2 12 12 17 22 12" />
    </svg>
  );
}*/

function SliderSvg(props: React.ComponentProps<"svg">) {
  return (
    <svg {...props} focusable="false" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth={1.5} strokeLinecap="round" strokeLinejoin="round">
      <line x1="4" y1="21" x2="4" y2="14" />
      <line x1="4" y1="10" x2="4" y2="3" />
      <line x1="12" y1="21" x2="12" y2="12" />
      <line x1="12" y1="8" x2="12" y2="3" />
      <line x1="20" y1="21" x2="20" y2="16" />
      <line x1="20" y1="12" x2="20" y2="3" />
      <line x1="1" y1="14" x2="7" y2="14" />
      <line x1="9" y1="8" x2="15" y2="8" />
      <line x1="17" y1="16" x2="23" y2="16" />
    </svg>
  );
}

/*
function ShieldSvg(props: React.ComponentProps<"svg">) {
  return (
    <svg {...props} viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth={1.5} strokeLinecap="round" strokeLinejoin="round">
      <path d="M12 22s8-4 8-10V5l-8-3-8 3v7c0 6 8 10 8 10z" />
    </svg>
  );
}
 */

function CloudSvg(props: React.ComponentProps<"svg">) {
  return (
    <svg {...props} viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth={1.5} strokeLinecap="round" strokeLinejoin="round">
      <path d="M2.25 15a4.5 4.5 0 0 0 4.5 4.5H18a3.75 3.75 0 0 0 1.332-7.257 3 3 0 0 0-3.758-3.848 5.25 5.25 0 0 0-10.233 2.33A4.502 4.502 0 0 0 2.25 15z" />
    </svg>
  );
}

/* ── Feature list ── */

type FeatureItem = {
  title: string;
  Svg: React.ComponentType<React.ComponentProps<"svg">>;
  description: ReactNode;
};

const FeatureList: FeatureItem[] = [
  {
      title: "Fine-Grained Scheduling",
      Svg: SliderSvg,
      description: (
          <>
              Configure scheduling globally or via annotations at namespace or workload level.
              Supports flexible scheduling for multi-tenant clusters and teams across timezones.
          </>
      ),
  },
  {
      title: "Flexible Time Formats",
      Svg: ClockSvg,
      description: (
          <>
              Define scaling windows as recurring schedules (like Mon–Fri), RFC3339 timespans,
              or always/never rules. Treat time as a scaling dimension that best fits your
              infrastructure usage
          </>
      ),
  },
  {
      title: "Cloud Agnostic",
      Svg: CloudSvg,
      description: (
          <>
              Built for any Kubernetes distribution across AWS, GCP, Azure, and on-premises environments.
              Fully supports all Kubernetes resources and popular CRDs like KEDA, Prometheus, and Argo.
          </>
      ),
  }
];
/* ── Sub-components ── */

function Feature({title, Svg, description, idx}: FeatureItem & { idx: number }) {
    const delays = ["animate-delay-0", "animate-delay-200", "animate-delay-400"];
    return (
        <div
            className={clsx("col col--4", styles.col, "animate-fade-down animate-once", delays[idx] ?? "animate-delay-0")}>
            <div className="text--center">
                <div className={styles.iconWrap}>
                    <Svg className={styles.featureSvg} role="img" aria-label={title}/>
                </div>
            </div>
            <div className="text--center padding-horiz--md">
        <Heading as="h3">{title}</Heading>
        <p className={styles.description}>{description}</p>
      </div>
    </div>
  );
}

/* ── Carousel arrow icon ── */

function ChevronLeft(props: React.ComponentProps<"svg">) {
  return (
    <svg {...props} viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth={2.5} strokeLinecap="round" strokeLinejoin="round">
      <polyline points="15 18 9 12 15 6" />
    </svg>
  );
}

function ChevronRight(props: React.ComponentProps<"svg">) {
  return (
    <svg {...props} viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth={2.5} strokeLinecap="round" strokeLinejoin="round">
      <polyline points="9 18 15 12 9 6" />
    </svg>
  );
}

/* ── Section ── */

export default function KubeDownscalerFeatures(): ReactNode {
  const [activeIdx, setActiveIdx] = useState(0);
  const total = FeatureList.length;

  return (
    <>
      <section className={styles.features}>
        <div className="container">

          {/* ── Desktop: all 3 cards in a row ── */}
          <div className={clsx("row", styles.desktopRow)}>
            {FeatureList.map((props, idx) => (
              <Feature key={props.title} idx={idx} {...props} />
            ))}
          </div>

          {/* ── Mobile: carousel (one card at a time) ── */}
          <div className={styles.carousel}>
            <button
              className={styles.arrowBtn}
              onClick={() => setActiveIdx(i => Math.max(0, i - 1))}
              disabled={activeIdx === 0}
              aria-label="Previous card"
            >
              <ChevronLeft className={styles.arrowIcon} />
            </button>

            <div className={styles.carouselTrack}>
              {FeatureList.map((props, idx) => (
                <div
                  key={props.title}
                  className={clsx(styles.carouselSlide, idx === activeIdx && styles.carouselSlideActive)}
                  aria-hidden={idx !== activeIdx}
                >
                  <Feature idx={idx} {...props} />
                </div>
              ))}
            </div>

            <button
              className={styles.arrowBtn}
              onClick={() => setActiveIdx(i => Math.min(total - 1, i + 1))}
              disabled={activeIdx === total - 1}
              aria-label="Next card"
            >
              <ChevronRight className={styles.arrowIcon} />
            </button>
          </div>

          {/* ── Mobile: dot indicators ── */}
          <div className={styles.dots}>
            {FeatureList.map((_, i) => (
              <button
                key={i}
                className={clsx(styles.dot, i === activeIdx && styles.dotActive)}
                onClick={() => setActiveIdx(i)}
                aria-label={`Go to card ${i + 1}`}
              />
            ))}
          </div>

        </div>
      </section>

      {/* ── Buttons below the section ── */}
      <div className="container">
        <div className="flex flex-col sm:flex-row gap-3 mt-10 sm:mt-6 mb-8 animate-fade-down animate-once animate-delay-700 justify-center items-center">
          <Button name="Get Started" to="/guides/getting-started" className={styles.ctaButton} primary />
          <GitHubButton className={styles.ctaButton} primary />
        </div>
      </div>
    </>
  );
}
