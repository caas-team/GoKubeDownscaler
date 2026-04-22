import React from "react";
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
  "animate-delay-1250",
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
      className={`animate-fade-down flex flex-col items-center text-center ${className}`}
    >
      <div className="flex justify-center mb-2">
        <Link href={href}>
          <Svg
            className="h-14 w-14 sm:h-20 sm:w-20 md:h-32 md:w-32 lg:h-32 lg:w-32 xl:h-40 xl:w-40"
            aria-label={title}
            role="img"
          />
        </Link>
      </div>
      <div className="px-1 w-full">
        <Heading
          as="h2"
          className="select-none text-sm sm:text-base md:text-xl lg:text-lg xl:text-xl"
        >
          {title}
        </Heading>
        {/* Desktop only: plain text list */}
        <p className="hidden sm:block text-xs sm:text-sm md:text-base lg:text-sm xl:text-base leading-relaxed">
          {supportedResources.join(", ")}
        </p>
      </div>
    </div>
  );
}

export function SupportedResources(): JSX.Element {
  const allPills = SupportedResourceGroupList.flatMap((g) => g.supportedResources);
  return (
    <div>
      <Heading className="block w-full text-center pt-10 md:pt-16 select-none" as="h1">
        Supported Resources
      </Heading>
      <section className="px-4 md:px-12 lg:px-8 pb-10 md:pb-16 pt-6 md:pt-8 w-full">
        <div className="mx-auto max-w-7xl w-full">
          <div className="grid grid-cols-3 md:grid-cols-3 lg:grid-cols-6 gap-3 md:gap-8 lg:gap-6 items-start">
            {SupportedResourceGroupList.map((props, idx) => (
              <SupportedResourceGroup
                className={delayClasses[idx] || ""}
                key={idx}
                {...props}
              />
            ))}
          </div>

          {/* Mobile-only: unified pill cloud below the logo grid */}
          <div className="flex sm:hidden flex-wrap justify-center gap-1.5 mt-5 animate-fade-down animate-once animate-delay-500">
            {allPills.map((r) => (
              <span
                key={r}
                className="inline-block rounded-full px-2.5 py-1 text-[0.72rem] font-medium leading-tight
                  bg-magenta/90 text-white/90
                  dark:bg-magenta/40 dark:text-pink-100"
              >
                {r}
              </span>
            ))}
          </div>
        </div>
      </section>
    </div>
  );
}
