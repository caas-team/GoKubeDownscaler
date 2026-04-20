import React from "react";
import Heading from "@theme/Heading";
import styles from "./styles.module.css";

export default function ProjectDescription(): JSX.Element {
  return (
    <section className={styles.section}>
      <div className={styles.inner}>
        <Heading as="h2" className={`${styles.headline} animate-fade-down animate-once animate-delay-0`}>
            Smart Kubernetes Autoscaling Powered By Schedules
        </Heading>
        <p className={`${styles.body} animate-fade-down animate-once animate-delay-200`}>
            GoKubeDownscaler acts as a horizontal autoscaler that reduces cloud costs by
            keeping workloads running only when needed. It scales workloads down
            during low-usage periods (like nights, weekends, and holidays) using user-defined schedules
        </p>
      </div>
    </section>
  );
}
