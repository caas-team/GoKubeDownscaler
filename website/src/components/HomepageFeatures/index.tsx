import clsx from "clsx";
import Heading from "@theme/Heading";
import styles from "./styles.module.css";
import { useState } from "react";

type FeatureItem = {
  title: string;
  Svg: React.ComponentType<React.ComponentProps<"svg">>;
  href: string;
  description: JSX.Element;
  selected?: boolean;
  onClick?: () => void;
};

const FeatureList: FeatureItem[] = [
  {
    title: "Kubernetes",
    Svg: require("@site/static/img/Kubernetes.svg").default,
    href: "https://kubernetes.io/",
    description: (
      <>
        batch/CronJob, apps/DeamonSet, apps/Deployment,
        autoscaling/HorizontalPodAutoscaler, batch/Job,
        policy/PodDisruptionBudget, apps/StatefulSet
      </>
    ),
  },
  {
    title: "Prometheus",
    Svg: require("@site/static/img/Prometheus.svg").default,
    href: "https://prometheus.io/",
    description: <>monitoring.coreos.com/Prometheus</>,
  },
  {
    title: "Argo",
    Svg: require("@site/static/img/Argo.svg").default,
    href: "https://argoproj.github.io/",
    description: <>argoproj.io/Rollout</>,
  },
  {
    title: "Keda",
    Svg: require("@site/static/img/Keda.svg").default,
    href: "https://keda.sh/",
    description: <>keda.sh/ScaledObject</>,
  },
  {
    title: "Zalando",
    Svg: require("@site/static/img/Zalando.svg").default,
    href: "https://zalando.com/",
    description: <>zalando.org/Stack</>,
  },
];

function Feature({
  title,
  Svg,
  href,
  description,
  selected,
  onClick,
}: FeatureItem) {
  return (
    <div
      className={clsx(
        "col",
        styles.feature,
        selected ? "col--12" + styles.selectedFeature : ""
      )}
      onClick={() => {
        if (!selected) onClick();
      }}
    >
      <div className={clsx("text--center")}>
        {/*<a href={href}>*/}
        <Svg className={styles.image} />
        {/*</a>*/}
      </div>
      <div className="text--center padding-horiz--md">
        <Heading as="h3">{title}</Heading>
        {selected && <p>{description}</p>}
      </div>
    </div>
  );
}

export default function HomepageFeatures(): JSX.Element {
  const [selected, setSelected] = useState("");

  return (
    <>
      <h1 className={styles.heading}>Supported Resources</h1>
      <section className={styles.features}>
        <div className="container">
          <div className="row">
            {FeatureList.map((props, idx) => (
              <Feature
                key={idx}
                {...props}
                onClick={() => setSelected(props.title)}
                selected={props.title == selected}
              />
            ))}
          </div>
        </div>
      </section>
    </>
  );
}
