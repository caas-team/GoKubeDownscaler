import useDocusaurusContext from "@docusaurus/useDocusaurusContext";
import Layout from "@theme/Layout";
import { SupportedResources } from "@site/src/components/Homepage/SupportedResources";
import { Button } from "../components/Basic/Button";
import * as KubedownscalerNameSVG from "@site/static/img/kubedownscaler-name-light.svg";
import * as KubedownscalerSVG from "@site/static/img/kubedownscaler.svg";
import Heading from "@theme/Heading";

function HomepageHeader() {
  const { siteConfig } = useDocusaurusContext();
  return (
    <div className="useTailwind relative overflow-x-hidden overflow-y-visible h-fit pb-35 xl:pb-0">
      <div className="transform bg-magenta -skew-y-6 xl:hidden h-full w-full absolute top-0 origin-top-left" />
      <header className="select-none text-white bg-magenta items-center flex py-16 px-8 overflow-hidden relative text-center h-fit">
        <div className="px-4 w-full flex flex-col items-center justify-center">
          <KubedownscalerNameSVG.default className="h-10 hidden xl:block" />
          <div className="xl:hidden flex flex-col justify-center">
            <KubedownscalerSVG.default className="h-30" />
            <Heading as="h1" className="text-5xl">
              {siteConfig.title}
            </Heading>
          </div>
          <p className="text-2xl">{siteConfig.tagline}</p>
          <div className="flex justify-center space-x-3">
            <Button name="Docs" to="/docs" className="w-32" />
            <Button name="Guides" to="/guides" className="w-32" />
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
      <HomepageHeader />
      <main>
        <SupportedResources />
      </main>
    </Layout>
  );
}
