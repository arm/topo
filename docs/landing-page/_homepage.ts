export type HomepageLink = {
  label: string;
  to: string;
  external?: boolean;
  variant: "primary" | "secondary";
};

export type BottomCard = {
  label: string;
  title: string;
  description: string;
  to: string;
  cta: string;
  variant: "primary" | "secondary";
};

export type CodeExample = {
  label: string;
  title: string;
  description: string;
  code: string;
};

export const homepageContent = {
  meta: {
    title: "Topo",
    description:
      "Discover, configure, and deploy projects to SSH targets.",
  },
  headline: "Bootstrap and accelerate Arm Linux development with Topo",
  lead: "Discover container-based projects which unlock the potential of your Arm device. Configure them for your use case. Deploy and iterate over SSH.",
  heroLinks: [
    {
      label: "Overview",
      to: "/introduction",
      variant: "primary",
    },
    {
      label: "View repository",
      to: "https://github.com/arm/topo",
      external: true,
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
        code: `topo health --target user@target.example`,
      },
      {
        label: "Discover",
        title: "Find projects that fit.",
        description:
          "Match projects to the capabilities available on the target device.",
        code: `topo projects --target user@target.example`,
      },
      {
        label: "Configure",
        title: "Prepare the project.",
        description: "Copy and configure a project on the host.",
        code: `topo clone https://github.com/Arm-Examples/topo-welcome.git`,
      },
      {
        label: "Deploy",
        title: "Ship over SSH.",
        description:
          "Configure, build, transfer, and start the Compose project on the target.",
        code: `topo deploy --target user@target.example`,
      },
    ] as CodeExample[],
  },
  bottomCards: [
    {
      label: "Getting started",
      title: "Install and deploy with Topo",
      description:
        "Install Topo and deploy your first project.",
      to: "/introduction/install",
      cta: "Install Topo",
      variant: "secondary",
    },
    {
      label: "Project specification",
      title: "Author your own Topo Projects",
      description:
        "Add Topo metadata to a standard Compose project to enable hardware compatibility matching and configuration.",
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
