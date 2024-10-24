import clsx from "clsx";
import Heading from "@theme/Heading";

type SupportedResourceGroupProps = {
  title: string;
  Svg: React.ComponentType<React.ComponentProps<"svg">>;
  href: string;
  supportedResources: string[];
};

const SupportedResourceGroupList: SupportedResourceGroupProps[] = [
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

function SupportedResourceGroup({
  title,
  Svg,
  href,
  supportedResources,
}: SupportedResourceGroupProps) {
  return (
    <div className="flex-grow max-w-full px-4 w-full xl:flex-1">
      <div className="text-center">
        <a href={href}>
          <Svg className="h-40 w-40" />
        </a>
      </div>
      <div className="text-center px-4">
        <Heading as="h3" className="select-none">
          {title}
        </Heading>
        <p>{supportedResources.join(", ")}</p>
      </div>
    </div>
  );
}

export function SupportedResources(): JSX.Element {
  return (
    <>
      <h1 className="block w-full text-center pt-8 select-none">
        Supported Resources
      </h1>
      <section className="flex items-center p-8 w-full">
        <div className="mx-auto max-w-6xl px-4 w-full">
          <div className="flex flex-wrap -mx-4 space-x-4">
            {SupportedResourceGroupList.map((props, idx) => (
              <SupportedResourceGroup key={idx} {...props} />
            ))}
          </div>
        </div>
      </section>
    </>
  );
}
