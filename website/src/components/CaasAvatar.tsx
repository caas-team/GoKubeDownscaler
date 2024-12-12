import Link from "@docusaurus/Link";
import CaasLogo from "/img/CaaS-Logo.svg";

export default function CaasAvatar(): JSX.Element {
  const name = "CaaS@DTIT";
  const description = "The container as a service team at Deutsch Telekom IT";
  const link = "https://github.com/caas-team";

  return (
    <div className="my-4">
      <div className="mx-auto max-w-6xl px-4 w-full gap-4 flex">
        <Link
          className="w-16 overflow-hidden h-16 block rounded-full"
          href={link}
        >
          <CaasLogo />
        </Link>
        <div className="flex flex-auto flex-col justify-center text-inherit">
          <Link target="_blank" href={link}>
            <div className="font-bold text-base">{name}</div>
          </Link>
          <small className="mt-1">{description}</small>
        </div>
      </div>
    </div>
  );
}
