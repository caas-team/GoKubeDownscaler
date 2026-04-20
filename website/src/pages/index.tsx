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
          {/* Logo */}
          <KubedownscalerSVG.default className="animate-fade-down h-28 sm:h-36 md:h-44" />
          {/* Name */}
          <Heading
            as="h1"
            className="animate-fade-down text-[clamp(1.75rem,6vw,3.5rem)] font-bold m-0"
            style={{ fontFamily: "'Poppins', sans-serif" }}
          >
            {siteConfig.title}
          </Heading>
          {/* Subtitle */}
          <p className="animate-fade-down text-lg sm:text-xl md:text-2xl lg:text-3xl max-w-4xl m-0">
              Reduce Kubernetes Costs By Scaling Workloads Down After Hours
          </p>
          {/* CTA buttons */}
          <div className="animate-fade-down flex justify-center gap-3 flex-col sm:flex-row">
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
        {/* Open Graph */}
        <meta property="og:type" content="website" />
        <meta property="og:url" content="https://caas-team.github.io/GoKubeDownscaler/" />
        <meta property="og:site_name" content="GoKubeDownscaler" />
        <meta property="og:title" content="GoKubeDownscaler: Reduce Kubernetes Costs Off-Hours" />
        <meta property="og:description" content="GoKubeDownscaler is a horizontal autoscaler that scales Kubernetes workloads down during off-hours like nights, weekend, holidays to reduce cloud costs." />
        <meta property="og:image" content="https://caas-team.github.io/GoKubeDownscaler/img/social-preview.png" />
        <meta property="og:image:width" content="1280" />
        <meta property="og:image:height" content="640" />
        <meta property="og:image:alt" content="GoKubeDownscaler — Kubernetes Scheduled Autoscaler" />
        {/* Twitter Card */}
        <meta name="twitter:card" content="summary_large_image" />
        <meta name="twitter:title" content="GoKubeDownscaler: Reduce Kubernetes Costs Off-Hours" />
        <meta name="twitter:description" content="GoKubeDownscaler is a horizontal autoscaler that scales Kubernetes workloads down during off-hours like nights, weekend, holidays to reduce cloud costs." />
        <meta name="twitter:image" content="https://caas-team.github.io/GoKubeDownscaler/img/social-preview.png" />
        {/* Canonical */}
        <link rel="canonical" href="https://caas-team.github.io/GoKubeDownscaler/" />
        {/* FAQPage structured data */}
        <script type="application/ld+json">{JSON.stringify({
          "@context": "https://schema.org",
          "@type": "FAQPage",
          mainEntity: [
            {
              "@type": "Question",
              name: "What is GoKubeDownscaler (kube-downscaler)?",
              acceptedAnswer: {
                "@type": "Answer",
                text: "GoKubeDownscaler (also known as a kube-downscaler) is an open-source Kubernetes horizontal autoscaler that automatically scales workloads down to zero during off-hours — such as nights, weekends, or holidays — to significantly reduce cloud infrastructure costs.",
              },
            },
            {
              "@type": "Question",
              name: "How does GoKubeDownscaler reduce cloud costs?",
              acceptedAnswer: {
                "@type": "Answer",
                text: "By scaling Kubernetes workloads down to zero during periods of low usage (e.g. after office hours, weekends, holidays), cluster nodes are also freed up, reducing the total number of running VMs and lowering your cloud bill.",
              },
            },
            {
              "@type": "Question",
              name: "Does GoKubeDownscaler work with EKS, GKE, and AKS?",
              acceptedAnswer: {
                "@type": "Answer",
                text: "Yes. GoKubeDownscaler works with any standard Kubernetes cluster, including Amazon EKS, Google GKE, Azure AKS, and self-managed clusters.",
              },
            },
            {
              "@type": "Question",
              name: "Is GoKubeDownscaler free to use?",
              acceptedAnswer: {
                "@type": "Answer",
                text: "Yes. GoKubeDownscaler is a free, open-source project licensed under the Apache 2.0 license. It is available on GitHub at https://github.com/caas-team/GoKubeDownscaler.",
              },
            },
            {
              "@type": "Question",
              name: "How do I install GoKubeDownscaler?",
              acceptedAnswer: {
                "@type": "Answer",
                text: "The easiest way to install GoKubeDownscaler is via Helm: run `helm upgrade -i gokubedownscaler oci://ghcr.io/caas-team/charts/go-kube-downscaler`. You can also install it using raw Kubernetes manifests.",
              },
            },
            {
              "@type": "Question",
              name: "What Kubernetes resources can GoKubeDownscaler scale?",
              acceptedAnswer: {
                "@type": "Answer",
                text: "GoKubeDownscaler supports scaling Deployments, StatefulSets, DaemonSets, CronJobs, HorizontalPodAutoscalers, PodDisruptionBudgets, Jobs, Argo Rollouts, KEDA ScaledObjects, Zalando Stacks, Prometheus instances, and GitHub Actions AutoscalingRunnerSets.",
              },
            },
            {
              "@type": "Question",
              name: "How can I scale Kubernetes workloads to zero at night?",
              acceptedAnswer: {
                  "@type": "Answer",
                  text: "You can scale Kubernetes workloads to zero at night by using a scheduled autoscaler like GoKubeDownscaler. It automatically reduces replicas to zero during off-hours, helping eliminate unnecessary resource usage and costs.",
              },
            },
            {
              "@type": "Question",
              name: "Can Kubernetes scale to zero automatically?",
              acceptedAnswer: {
                  "@type": "Answer",
                  text: "By default, Kubernetes Horizontal Pod Autoscaler (HPA) does not support scaling to zero because it requires running pods to collect metrics. Tools like GoKubeDownscaler enable scheduled scale-to-zero behavior for cost optimization.",
              },
            },
            {
              "@type": "Question",
              name: "How do I reduce Kubernetes cloud costs?",
              acceptedAnswer: {
                  "@type": "Answer",
                  text: "One of the most effective ways to reduce Kubernetes cloud costs is to scale workloads down during periods of low usage. GoKubeDownscaler automates this by scheduling scale-down events, freeing cluster resources and reducing the number of running nodes.",
              },
            },
            {
              "@type": "Question",
              name: "What is the difference between HPA and kube-downscaler?",
              acceptedAnswer: {
                  "@type": "Answer",
                  text: "The Kubernetes Horizontal Pod Autoscaler (HPA) scales workloads dynamically based on metrics like CPU or memory usage, while kube-downscaler scales workloads based on time schedules. This makes it ideal for predictable off-hours cost savings.",
              },
            },
            {
              "@type": "Question",
              name: "Does GoKubeDownscaler need cluster wide permissions?",
              acceptedAnswer: {
                "@type": "Answer",
                text: "Not necessarily. GoKubeDownscaler can be installed cluster-wide with permissions to manage all namespaces, or it can be installed with limited permissions to manage specific namespaces or workloads. This flexibility allows you to implement cost-saving schedules without granting unnecessary access.",
              },
            },
            {
              "@type": "Question",
              name: "Does GoKubeDownscaler (kube-downscaler) automatically handle daylight saving time (DST)?",
              acceptedAnswer: {
                "@type": "Answer",
                text: "Yes, GoKubeDownscaler automatically adjusts for daylight saving time (DST) when scheduling scale-down events. It uses the IANA timezone database to ensure that workloads are scaled down at the correct local times, even as DST changes occur.",
              },
            }
          ],
        })}</script>
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
