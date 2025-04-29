import useDocusaurusContext from "@docusaurus/useDocusaurusContext";
import Layout from "@theme/Layout";
import { SupportedResources } from "@site/src/components/Homepage/SupportedResources";
import { Button } from "../components/Basic/Button";
import * as KubedownscalerNameSVG from "@site/static/img/kubedownscaler-name-light.svg";
import * as KubedownscalerSVG from "@site/static/img/kubedownscaler.svg";
import Heading from "@theme/Heading";
import Head from "@docusaurus/Head";

function HomepageHeader() {
  const { siteConfig } = useDocusaurusContext();
  return (
    <div className="relative overflow-x-hidden overflow-y-visible h-fit pb-35 xl:pb-0">
      <div className="transform bg-magenta -skew-y-6 xl:hidden h-full w-full absolute top-0 origin-top-left" />
      <header className="select-none text-white bg-magenta items-center flex py-16 px-8 overflow-hidden relative text-center h-fit">
        <div className="px-4 w-full flex flex-col items-center justify-center">
          <KubedownscalerNameSVG.default className="animate-fade-down h-10 hidden xl:block" />
          <div className="animate-fade-down xl:hidden flex flex-col justify-center">
            <KubedownscalerSVG.default className="h-30" />
            <Heading as="h1" className="text-[clamp(1.25rem,8vw,3rem)]">
              {siteConfig.title}
            </Heading>
          </div>
          <p className="animate-fade-down text-2xl">{siteConfig.tagline}</p>
          <div className="flex justify-center space-x-0 sm:space-x-3 sm:flex-row space-y-3 sm:space-y-0 flex-col">
            <Button
              name="Get Started"
              to="/guides/getting-started"
              className="w-52"
            />
            <Button name="Documentation" to="/docs" className="w-52" />
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
        <title>
          {siteConfig.title}: {siteConfig.tagline}
        </title>
      </Head>
      <HomepageHeader />
      <main>
        <SupportedResources />
      </main>
    </Layout>
  );
}
