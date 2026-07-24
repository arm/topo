export type HomepageLink = {
  label: string;
  to?: string;
  href?: string;
  variant: "primary" | "secondary";
};

export type BottomCard = {
  label: string;
  title: string;
  description: string;
  to?: string;
  href?: string;
  disabled?: boolean;
  cta: string;
  variant: "primary" | "secondary";
};

export type CodeExample = {
  label: string;
  title: string;
  description: string;
  language: string;
  code: string;
};

export const homepageContent = {
  meta: {
    title: "Topo",
    description:
      "Discover Arm device capabilities and deploy containerised software over SSH.",
  },
  headline: "Build on your laptop. Run on Arm with Topo.",
  lead: "Topo discovers Arm device features, automatically filters supported projects, and simplifies configuration and deployment of containerised software over SSH.",
  heroLinks: [
    {
      label: "Overview",
      to: "/introduction",
      variant: "primary",
    },
    {
      label: "View repository",
      href: "https://github.com/arm/topo",
      variant: "secondary",
    },
  ] as HomepageLink[],
  codeExamples: {
    eyebrow: "Topo workflow",
    title: "Get your project running in seconds",
    subtitle:
      "Check device features and health, find compatible projects, and deploy over SSH without replacing your existing container workflow.",
    items: [
      {
        label: "Check",
        title: "Know the target is ready.",
        description:
          "Verify the host, SSH connection, target, and hardware before deploying.",
        language: "shell",
        code: `topo health --target pi@raspberrypi`,
      },
      {
        label: "Discover",
        title: "Find projects that fit.",
        description:
          "Match projects to the capabilities available on the target device.",
        language: "shell",
        code: `topo projects --target pi@raspberrypi`,
      },
      {
        label: "Deploy",
        title: "Ship over SSH.",
        description:
          "Configure, build, transfer, and start the Compose project on the target.",
        language: "shell",
        code: `topo deploy --target pi@raspberrypi`,
      },
    ] as CodeExample[],
  },
  bottomCards: [
    {
      label: "Getting started",
      title: "Install and deploy with Topo",
      description:
        "Install Topo, check a target, and deploy your first project.",
      to: "/introduction/install",
      cta: "Install Topo",
      variant: "secondary",
    },
    {
      label: "Project specification",
      title: "Build hardware-aware projects",
      description:
        "Add Topo metadata to a standard Compose project.",
      to: "/project-specification",
      cta: "Read the specification",
      variant: "secondary",
    },
    {
      label: "Development",
      title: "Contribute to Topo",
      description:
        "Work on Topo itself and follow the contributor workflow.",
      to: "/development",
      cta: "Open the guide",
      variant: "secondary",
    },
  ] as BottomCard[],
};
