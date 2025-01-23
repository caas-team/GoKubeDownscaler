import useDocusaurusContext from "@docusaurus/useDocusaurusContext";
import Layout from "@theme/Layout";
import { SupportedResources } from "@site/src/components/Homepage/SupportedResources";
import Heading from "@theme/Heading";
import { Button } from "../components/Basic/Button";
import ThemedImage from "@theme/ThemedImage";
import useBaseUrl from "@docusaurus/useBaseUrl";

function HomepageHeader() {
  const { siteConfig } = useDocusaurusContext();
  return (
    <header className="select-none p-8 text-white bg-magenta items-center flex lg:py-16 lg:px-8 overflow-hidden relative text-center">
      <div className="mx-auto max-w-6xl px-4 w-full">
        <div className="items-center h-64">
          <ThemedImage
            alt="Kubevela Logo"
            className="h-4/5"
            sources={{
              light: useBaseUrl("img/kubedownscaler-dark.svg"),
              dark: useBaseUrl("img/kubedownscaler-light.svg"),
            }}
          />
        </div>
        <Heading as="h1" className="text-5xl">
          {siteConfig.title}
        </Heading>
        <p className="text-2xl">{siteConfig.tagline}</p>
        <div className="flex items-center justify-center space-x-3">
          <Button name="Docs" to="/docs" className="w-32" />
          <Button name="Guides" to="/guides" className="w-32" />
        </div>
      </div>
    </header>
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
