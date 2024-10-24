import clsx from "clsx";
import Link from "@docusaurus/Link";
import useDocusaurusContext from "@docusaurus/useDocusaurusContext";
import Layout from "@theme/Layout";
import { SupportedResources } from "@site/src/components/Homepage/SupportedResources";
import Heading from "@theme/Heading";

function HomepageHeader() {
  const { siteConfig } = useDocusaurusContext();
  return (
    <header className="select-none p-8 text-white bg-magenta items-center flex lg:py-16 lg:px-8 overflow-hidden relative text-center">
      <div className="mx-auto max-w-6xl px-4 w-full">
        <Heading as="h1" className="text-5xl">
          {siteConfig.title}
        </Heading>
        <p className="text-2xl">{siteConfig.tagline}</p>
        <div className="flex items-center justify-center">
          <Link
            className="bg-gray-100 hover:bg-gray-200 active:bg-gray-300 border border-solid border-gray-100 hover:border-gray-200 active:border-gray-300 rounded-md cursor-pointer text-xl font-bold py-2 px-8 text-center duration-200 transition-colors select-none whitespace-nowrap text-slate-900 hover:text-slate-900 hover:no-underline"
            to="/docs"
          >
            Documentation
          </Link>
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
