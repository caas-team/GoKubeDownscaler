import Heading from "@theme/Heading";
import * as PrometheusSVG from "@site/static/img/Prometheus.svg";
import * as ArgoSVG from "@site/static/img/Argo.svg";
import * as KubernetesSVG from "@site/static/img/Kubernetes.svg";
import * as KedaSVG from "@site/static/img/Keda.svg";
import * as ZalandoSVG from "@site/static/img/Zalando.svg";
import Link from "@docusaurus/Link";

const SupportedResourceGroupList: SupportedResourceGroupProps[] = [
  {
    title: "Prometheus",
    Svg: PrometheusSVG.default,
    href: "https://prometheus.io/",
    supportedResources: ["Prometheuses"],
  },
  {
    title: "Argo",
    Svg: ArgoSVG.default,
    href: "https://argoproj.github.io/",
    supportedResources: ["Rollouts"],
  },
  {
    title: "Kubernetes",
    Svg: KubernetesSVG.default,
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
    Svg: KedaSVG.default,
    href: "https://keda.sh/",
    supportedResources: ["ScaledObjects"],
  },
  {
    title: "Zalando",
    Svg: ZalandoSVG.default,
    href: "https://opensource.zalando.com/",
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
    <div className="max-w-full px-4 pb-8 w-full xl:flex-1">
      <div className="flex justify-center mb-2">
        <Link href={href}>
          <Svg className="h-40 w-40" />
        </Link>
      </div>
      <div className="text-center px-4 max-w-64 mx-auto">
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
    <div className="useTailwind">
      <Heading className="block w-full text-center pt-8 select-none" as="h1">
        Supported Resources
      </Heading>
      <section className="flex items-center p-8 w-full">
        <div className="mx-auto max-w-6xl px-4 w-full">
          <div className="flex flex-wrap -mx-4 space-x-4">
            {SupportedResourceGroupList.map((props, idx) => (
              <SupportedResourceGroup key={idx} {...props} />
            ))}
          </div>
        </div>
      </section>
    </div>
  );
}
