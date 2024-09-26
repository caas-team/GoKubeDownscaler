import clsx from "clsx";
import Heading from "@theme/Heading";
import styles from "./styles.module.css";

type FeatureItem = {
  title: string;
  Svg: React.ComponentType<React.ComponentProps<"svg">>;
  href: string;
  description: JSX.Element;
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

function Feature({ title, Svg, href, description }: FeatureItem) {
  return (
    <div className={clsx("col", styles.feature)}>
      <div className={clsx("text--center")}>
        <a href={href}>
          <Svg className={styles.image} />
        </a>
      </div>
      <div className="text--center padding-horiz--md">
        <Heading as="h3">{title}</Heading>
        <p>{description}</p>
      </div>
    </div>
  );
}

export default function HomepageFeatures(): JSX.Element {
  return (
    <section className={styles.features}>
      <h1 className={styles.heading}>Supported Resources</h1>
      <div className="container">
        <div className="row">
          {FeatureList.map((props, idx) => (
            <Feature key={idx} {...props} />
          ))}
        </div>
      </div>
    </section>
  );
}
