import CaasLogo from "/img/CaaS-Logo.svg";

export default function CaasAvatar(): JSX.Element {
  const name = "CaaS@DTIT";
  const description = "The Container as a Service Team at Deutsch Telekom IT";
  const link = "https://github.com/caas-team";

  return (
    <div className="section margin-vert--md">
      <div className="container avatar">
        <a
          className="avatar__photo-link avatar__photo avatar__photo--lg"
          href={link}
        >
          <CaasLogo />
        </a>
        <div className="avatar__intro">
          <a href={link}>
            <div className="avatar__name">{name}</div>
          </a>
          <small className="avatar__subtitle">{description}</small>
        </div>
      </div>
    </div>
  );
}
