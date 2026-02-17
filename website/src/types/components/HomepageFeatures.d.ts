interface SupportedResourceGroupProps {
  title: string;
  SvgLight: React.ComponentType<React.ComponentProps<"svg">>;
  SvgDark: React.ComponentType<React.ComponentProps<"svg">>;
  href: string;
  supportedResources: string[];
  className?: string;
}
