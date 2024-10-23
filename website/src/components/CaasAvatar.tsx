import CaasLogo from "/img/CaaS-Logo.svg";

export default function CaasAvatar(): JSX.Element {
  const name = "CaaS@DTIT";
  const description = "The Container as a Service Team at Deutsch Telekom IT";
  const link = "https://github.com/caas-team";

  return (
    <div className="my-4">
      <div className="mx-auto max-w-6xl px-4 w-full gap-4 flex">
        <a className="w-16 overflow-hidden h-16 block rounded-full" href={link}>
          <CaasLogo />
        </a>
        <div className="flex flex-auto flex-col justify-center text-inherit">
          <a href={link}>
            <div className="font-bold text-base">{name}</div>
          </a>
          <small className="mt-1">{description}</small>
        </div>
      </div>
    </div>
  );
}
