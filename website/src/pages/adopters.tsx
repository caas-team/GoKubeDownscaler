import Layout from "@theme/Layout";
import Heading from "@theme/Heading";
import Head from "@docusaurus/Head";
import * as KubedownscalerSVG from "@site/static/img/kubedownscaler.svg";
import { GitHubButton } from "../components/Basic/Button";

const ADOPTERS_ISSUE_URL =
    "https://github.com/caas-team/GoKubeDownscaler/issues/new?template=adopter.yaml";

function AdoptersHeader() {
    return (
        <div className="relative overflow-x-hidden overflow-y-visible">
            <div className="transform bg-magenta -skew-y-6 xl:hidden h-full w-full absolute top-0 origin-top-left" />
            <header className="select-none text-white bg-magenta items-center flex pt-10 pb-24 px-8 overflow-hidden relative text-center">
                <div className="px-4 w-full flex flex-col items-center justify-center gap-6">
                    <KubedownscalerSVG.default className="animate-fade-down h-28 sm:h-36 md:h-44" />
                    <Heading
                        as="h1"
                        className="animate-fade-down text-[clamp(1.25rem,4vw,2.5rem)] font-bold m-0"
                        style={{ fontFamily: "'Poppins', sans-serif" }}
                    >
                        Are you a GoKubeDownscaler adopter?
                    </Heading>
                    <p className="animate-fade-down text-lg sm:text-xl md:text-2xl max-w-2xl m-0">
                        Let us know! Add your organization to our adopters list.
                    </p>
                    <div className="animate-fade-down flex justify-center w-full max-w-xs">
                        <GitHubButton
                            href={ADOPTERS_ISSUE_URL}
                            label="Add Me as Adopter"
                            className="w-full"
                        />
                    </div>
                </div>
            </header>
        </div>
    );
}

export default function Adopters(): JSX.Element {
    return (
        <Layout
            title="Adopters"
            description="See which organizations use GoKubeDownscaler in production to reduce Kubernetes cloud costs. Are you an adopter? Let us know!"
        >
            <Head>
                <title>Adopters | GoKubeDownscaler</title>
                <meta
                    name="description"
                    content="See which organizations use GoKubeDownscaler in production to reduce Kubernetes cloud costs. Are you an adopter? Let us know!"
                />
                <meta
                    name="keywords"
                    content="GoKubeDownscaler adopters, kube-downscaler users, who uses GoKubeDownscaler, kubernetes downscaler production, kubernetes cost optimization users"
                />

                {/* Open Graph */}
                <meta property="og:type" content="website" />
                <meta property="og:url" content="https://kube-downscaler.io/adopters" />
                <meta property="og:site_name" content="GoKubeDownscaler" />
                <meta property="og:title" content="Adopters | GoKubeDownscaler" />
                <meta
                    property="og:description"
                    content="See which organizations use GoKubeDownscaler in production to reduce Kubernetes cloud costs. Are you an adopter? Let us know!"
                />
                <meta property="og:image" content="https://kube-downscaler.io/img/social-preview.png" />
                <meta property="og:image:width" content="1280" />
                <meta property="og:image:height" content="640" />
                <meta property="og:image:alt" content="GoKubeDownscaler — Kubernetes Scheduled Autoscaler" />

                {/* Twitter Card */}
                <meta name="twitter:card" content="summary_large_image" />
                <meta name="twitter:title" content="Adopters | GoKubeDownscaler" />
                <meta
                    name="twitter:description"
                    content="See which organizations use GoKubeDownscaler in production to reduce Kubernetes cloud costs. Are you an adopter? Let us know!"
                />
                <meta name="twitter:image" content="https://kube-downscaler.io/img/social-preview.png" />

                {/* Canonical */}
                <link rel="canonical" href="https://kube-downscaler.io/adopters" />

                {/* Structured data */}
                <script type="application/ld+json">{JSON.stringify({
                    "@context": "https://schema.org",
                    "@type": "WebPage",
                    "@id": "https://kube-downscaler.io/adopters/#webpage",
                    name: "GoKubeDownscaler Adopters",
                    url: "https://kube-downscaler.io/adopters",
                    description:
                        "Organizations using GoKubeDownscaler in production to reduce Kubernetes cloud costs by scaling workloads down after hours.",
                    isPartOf: { "@id": "https://kube-downscaler.io/#website" },
                    about: { "@id": "https://kube-downscaler.io/#software" },
                })}</script>
            </Head>
            <AdoptersHeader />
            <main className="container mx-auto px-4 py-12 text-center">
                <p className="text-xl text-gray-500 dark:text-gray-400">
                    Adopters list coming soon…
                </p>
            </main>
        </Layout>
    );
}
