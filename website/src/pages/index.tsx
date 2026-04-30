import useDocusaurusContext from "@docusaurus/useDocusaurusContext";
import Layout from "@theme/Layout";
import { SupportedResources } from "@site/src/components/Homepage/SupportedResources/SupportedResources.tsx";
import ProjectDescription from "@site/src/components/Homepage/ProjectDescription";
import KubeDownscalerFeatures from "@site/src/components/Homepage/KubeDownscalerFeatures";
import HowItWorks from "@site/src/components/Homepage/HowItWorks";
import FurtherCustomization from "@site/src/components/Homepage/FurtherCustomization";
import { Button, GitHubButton } from "../components/Basic/Button";
import * as KubedownscalerSVG from "@site/static/img/kubedownscaler.svg";
import Heading from "@theme/Heading";
import Head from "@docusaurus/Head";

function HomepageHeader() {
  const { siteConfig } = useDocusaurusContext();
  return (
    <div className="relative overflow-x-hidden overflow-y-visible">
      <div className="transform bg-magenta -skew-y-6 xl:hidden h-full w-full absolute top-0 origin-top-left" />
      <header className="select-none text-white bg-magenta items-center flex pt-10 pb-24 px-8 overflow-hidden relative text-center">
        <div className="px-4 w-full flex flex-col items-center justify-center gap-6">
          {/* Logo — Fixed height to prevent CLS, will-change for animation optimization */}
          <div className="h-28 sm:h-36 md:h-44 flex items-center justify-center" style={{ willChange: "transform" }}>
            <KubedownscalerSVG.default
              className="animate-fade-down h-full w-auto"
              role="img"
              aria-label="GoKubeDownscaler Logo - Kubernetes Cost Optimization Tool"
            />
          </div>
          {/* Name — Minimum height to prevent CLS */}
          <div style={{ willChange: "transform" }}>
            <Heading
              as="h1"
              className="animate-fade-down text-[clamp(1.75rem,6vw,3.5rem)] font-bold m-0"
              style={{ fontFamily: "'Poppins', sans-serif", minHeight: "3.5rem" }}
            >
              {siteConfig.title}
            </Heading>
          </div>
          {/* Subtitle — Minimum height to prevent CLS */}
          <div style={{ willChange: "transform" }}>
            <p className="animate-fade-down text-lg sm:text-xl md:text-2xl lg:text-3xl max-w-4xl m-0" style={{ minHeight: "2rem" }}>
                Reduce Kubernetes Costs By Scaling Workloads Down After Hours
            </p>
          </div>
          {/* CTA buttons — Container with min-height */}
          <div className="animate-fade-down flex justify-center gap-3 flex-col sm:flex-row" style={{ willChange: "transform", minHeight: "2.5rem" }}>
            <Button name="Get Started" to="/guides/getting-started" className="w-52" />
            <Button name="Documentation" to="/docs" className="w-52" />
            <GitHubButton className="hidden sm:block w-52" />
          </div>
        </div>
      </header>
    </div>
  );
}

export default function Home(): JSX.Element {
  const { siteConfig } = useDocusaurusContext();
  return (
    <Layout title={siteConfig.title} description={siteConfig.tagline}>
      <Head>
        <title>GoKubeDownscaler: Reduce Kubernetes Costs Off-Hours</title>
        <meta name="description" content="GoKubeDownscaler is a horizontal autoscaler that scales Kubernetes workloads down during off-hours like nights, weekend, holidays to reduce cloud costs." />
        <meta name="keywords" content="kube-downscaler, GoKubeDownscaler, kubernetes downscaler, kubernetes cost optimization, scale down kubernetes, kubernetes scheduled scaling, go-kube-downscaler, kubernetes autoscaler" />
        {/* Open Graph — page-specific (og:type, og:site_name, og:image:* are global in docusaurus.config.ts) */}
        <meta property="og:url" content="https://kube-downscaler.io/" />
        <meta property="og:title" content="GoKubeDownscaler: Reduce Kubernetes Costs Off-Hours" />
        <meta property="og:description" content="GoKubeDownscaler is a horizontal autoscaler that scales Kubernetes workloads down during off-hours like nights, weekend, holidays to reduce cloud costs." />
        {/* Twitter — page-specific (twitter:card is global in docusaurus.config.ts; twitter:image injected by Docusaurus from themeConfig.image) */}
        <meta name="twitter:title" content="GoKubeDownscaler: Reduce Kubernetes Costs Off-Hours" />
        <meta name="twitter:description" content="GoKubeDownscaler is a horizontal autoscaler that scales Kubernetes workloads down during off-hours like nights, weekend, holidays to reduce cloud costs." />
        <meta name="twitter:image" content="https://caas-team.github.io/GoKubeDownscaler/img/social-preview.png" />
      </Head>
      <HomepageHeader />
      <main>
        <ProjectDescription />
        <KubeDownscalerFeatures />
        <HowItWorks />
        <FurtherCustomization />
        <SupportedResources />
      </main>
    </Layout>
  );
}
