import React from "react";
import Heading from "@theme/Heading";
import {Button, GitHubButton} from "@site/src/components/Basic/Button";
import styles from "./styles.module.css";

const commands = [
  "kubectl annotate namespace spanish-namespace downscaler/uptime=\"Mon-Fri 09:00-17:00 Europe/Madrid\"",
  "",
  "kubectl annotate deploy istiod -n istio-system downscaler/exclude=\"true\"",
  "",
  "kubectl annotate namespace webapp-hotfixbranch downscaler/exclude-until=\"2027-07-29T21:30:00Z\"",
];

/** Renders a command string, highlighting everything from "downscaler/" onwards in red. */
function renderCommand(cmd: string): React.ReactNode {
  if (!cmd) return null;
  const idx = cmd.indexOf("downscaler/");
  if (idx === -1) return cmd;
  return (
    <>
      {cmd.slice(0, idx)}
      <span className={styles.annotationKey}>{cmd.slice(idx)}</span>
    </>
  );
}

export default function FurtherCustomization(): JSX.Element {
  return (
    <section className={styles.section}>
      <div className={styles.inner}>
        <Heading as="h2" className={`${styles.headline} animate-fade-down animate-once animate-delay-0`}>
          Needs Further Customization?
        </Heading>

        <p className={`${styles.body} animate-fade-down animate-once animate-delay-200`}>
            GoKubeDownscaler can be tuned far beyond a simple schedule.
            Use annotations at namespace or workload level to override global scheduling.
            For example, define per-timezone schedules for teams in other countries,
            or set up permanent or temporary exclusions. Discover all options in our documentation
        </p>

        <div className={`${styles.terminal} animate-fade-down animate-once animate-delay-500`}>
          {/* Title bar */}
          <div className={styles.titleBar}>
            <span className={styles.dot} style={{ background: "#ff5f57" }} />
            <span className={styles.dot} style={{ background: "#febc2e" }} />
            <span className={styles.dot} style={{ background: "#28c840" }} />
          </div>
          {/* Commands */}
          <pre className={styles.code}>
            <code>
              {commands.map((cmd, i) => (
                <React.Fragment key={i}>
                  <span className={styles.prompt}>$ </span>
                  <span className={styles.command}>{renderCommand(cmd)}</span>
                  {i < commands.length - 1 && "\n"}
                </React.Fragment>
              ))}
            </code>
          </pre>
        </div>

        <div className="flex flex-col sm:flex-row gap-3 mt-6 animate-fade-down animate-once animate-delay-700">
          <Button name="Get Started" to="/guides/getting-started" className="w-52" primary />
          <GitHubButton className="w-52" primary />
        </div>
      </div>
    </section>
  );
}
