import Heading from "@theme/Heading";
import * as PrometheusSVG from "@site/static/img/Prometheus.svg";
import * as ArgoSVG from "@site/static/img/Argo.svg";
import * as KubernetesSVG from "@site/static/img/Kubernetes.svg";
import * as KedaSVG from "@site/static/img/Keda.svg";
import * as ZalandoSVG from "@site/static/img/Zalando.svg";
import * as GithubLightSVG from "@site/static/img/Github-white.svg";
import * as GithubDarkSVG from "@site/static/img/Github.svg";
import { useColorMode } from "@docusaurus/theme-common";
import Link from "@docusaurus/Link";

const delayClasses = [
  "animate-delay-0",
  "animate-delay-250",
  "animate-delay-500",
  "animate-delay-750",
  "animate-delay-1000",
];

const SupportedResourceGroupList: SupportedResourceGroupProps[] = [
  {
    title: "Prometheus",
    SvgLight: PrometheusSVG.default,
    SvgDark: PrometheusSVG.default,
    href: "https://prometheus.io/",
    supportedResources: ["Prometheuses"],
  },
  {
    title: "Argo",
    SvgLight: ArgoSVG.default,
    SvgDark: ArgoSVG.default,
    href: "https://argoproj.github.io/",
    supportedResources: ["Rollouts"],
  },
  {
    title: "Kubernetes",
    SvgLight: KubernetesSVG.default,
    SvgDark: KubernetesSVG.default,
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
    SvgLight: KedaSVG.default,
    SvgDark: KedaSVG.default,
    href: "https://keda.sh/",
    supportedResources: ["ScaledObjects"],
  },
  {
    title: "Zalando",
    SvgLight: ZalandoSVG.default,
    SvgDark: ZalandoSVG.default,
    href: "https://opensource.zalando.com/",
    supportedResources: ["Stacks"],
  },
  {
    title: "Github Actions",
    SvgLight: GithubDarkSVG.default,
    SvgDark: GithubLightSVG.default,
    href: "https://docs.github.com/en/actions/concepts/runners/actions-runner-controller",
    supportedResources: ["AutoscalingRunnerSet"],
  },
];

function SupportedResourceGroup({
  title,
  SvgLight,
  SvgDark,
  href,
  supportedResources,
  className,
}: SupportedResourceGroupProps) {
  const { colorMode } = useColorMode();
  const Svg = colorMode === "dark" ? SvgDark : SvgLight;
  return (
    <div
      className={`animate-fade-down max-w-full px-4 pb-8 w-full xl:flex-1 ${className}`}
    >
      <div className="flex justify-center mb-2">
        <Link href={href}>
          <Svg className="h-40 w-40" aria-label={title} role="img" />
        </Link>
      </div>
      <div className="text-center px-4 max-w-64 mx-auto">
        <Heading as="h2" className="select-none">
          {title}
        </Heading>
        <p>{supportedResources.join(", ")}</p>
      </div>
    </div>
  );
}

export function SupportedResources(): JSX.Element {
  return (
    <div>
      <Heading className="block w-full text-center pt-8 select-none" as="h1">
        Supported Resources
      </Heading>
      <section className="flex items-center p-8 w-full">
        <div className="mx-auto max-w-7xl px-4 w-full">
          <div className="flex flex-wrap -mx-2 space-x-2">
            {SupportedResourceGroupList.map((props, idx) => (
              <SupportedResourceGroup
                className={delayClasses[idx] || ""}
                key={idx}
                {...props}
              />
            ))}
          </div>
        </div>
      </section>
    </div>
  );
}
