import clsx from "clsx";
import Heading from "@theme/Heading";
import styles from "./styles.module.css";

type FeatureItem = {
  title: string;
  Svg: React.ComponentType<React.ComponentProps<"svg">>;
  href: string;
  supportedResources: string[];
};

const FeatureList: FeatureItem[] = [
  {
    title: "Prometheus",
    Svg: require("@site/static/img/Prometheus.svg").default,
    href: "https://prometheus.io/",
    supportedResources: ["Prometheuses"],
  },
  {
    title: "Argo",
    Svg: require("@site/static/img/Argo.svg").default,
    href: "https://argoproj.github.io/",
    supportedResources: ["Rollouts"],
  },
  {
    title: "Kubernetes",
    Svg: require("@site/static/img/Kubernetes.svg").default,
    href: "https://kubernetes.io/",
    supportedResources: [
      "Deployment",
      "StatefulSet",
      "DeamonSet",
      "CronJob",
      "HorizontalPodAutoscaler",
      "PodDisruptionBudget",
      "Job",
    ],
  },
  {
    title: "Keda",
    Svg: require("@site/static/img/Keda.svg").default,
    href: "https://keda.sh/",
    supportedResources: ["ScaledObjects"],
  },
  {
    title: "Zalando",
    Svg: require("@site/static/img/Zalando.svg").default,
    href: "https://zalando.com/",
    supportedResources: ["Stacks"],
  },
];

function Feature({ title, Svg, href, supportedResources }: FeatureItem) {
  return (
    <div className={clsx("col", styles.feature)}>
      <div className={clsx("text--center")}>
        <a href={href}>
          <Svg className={styles.image} />
        </a>
      </div>
      <div className={"text--center padding-horiz--md"}>
        <Heading as="h3">{title}</Heading>
        <p>{supportedResources.join(", ")}</p>
      </div>
    </div>
  );
}

export default function HomepageFeatures(): JSX.Element {
  return (
    <>
      <h1 className={styles.heading}>Supported Resources</h1>
      <section className={styles.features}>
        <div className="container">
          <div className="row">
            {FeatureList.map((props, idx) => (
              <Feature key={idx} {...props} />
            ))}
          </div>
        </div>
      </section>
    </>
  );
}
