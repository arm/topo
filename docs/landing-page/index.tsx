import Link from "@docusaurus/Link";
import Layout from "@theme/Layout";
import type { ReactElement } from "react";

import { homepageContent } from "./_homepage";
import ExternalLinkIcon from "./external-link.svg";
import styles from "./index.module.css";

function joinClasses(...parts: Array<string | undefined | false>): string {
  return parts.filter(Boolean).join(" ");
}

export default function Home(): ReactElement {
  return (
    <Layout
      title={homepageContent.meta.title}
      description={homepageContent.meta.description}
    >
      <main className={styles.page}>
        <section className={styles.hero}>
          <div className={styles.heroContent}>
            <h1>{homepageContent.headline}</h1>
            <p className={styles.lead}>{homepageContent.lead}</p>
            <div className={styles.actions}>
              {homepageContent.heroLinks.map((link) => (
                <Link
                  key={link.label}
                  to={link.to}
                  target={link.external ? "_blank" : undefined}
                  rel={link.external ? "noreferrer" : undefined}
                  className={joinClasses(
                    `button button--${link.variant}`,
                    styles.button,
                  )}
                >
                  {link.label}
                  {link.external && (
                    <ExternalLinkIcon
                      className={styles.externalLinkIcon}
                      aria-hidden="true"
                      focusable="false"
                    />
                  )}
                </Link>
              ))}
            </div>
          </div>
          <div className={styles.heroVisual}>
            <img
              className={styles.heroDiagram}
              src="img/topo-overview.svg"
              alt="Topo deployment and development loop"
            />
          </div>
        </section>

        <section className={styles.bottom}>
          {homepageContent.bottomCards.map((card) => (
            <Link
              key={card.title}
              to={card.to}
              className={joinClasses(styles.bottomCard, styles.bottomCardLink)}
            >
              <div>
                <p className={styles.label}>{card.label}</p>
                <h4>{card.title}</h4>
                <p className={styles.small}>{card.description}</p>
              </div>
              <span
                className={joinClasses(
                  "button button--sm",
                  `button--${card.variant}`,
                )}
              >
                {card.cta}
              </span>
            </Link>
          ))}
        </section>

        <section className={styles.codeSection}>
          <div className={styles.codeIntro}>
            <p className={styles.codeEyebrow}>
              {homepageContent.codeExamples.eyebrow}
            </p>
            <h2>{homepageContent.codeExamples.title}</h2>
            <p>{homepageContent.codeExamples.subtitle}</p>
          </div>
          <div className={styles.codeGrid}>
            {homepageContent.codeExamples.items.map((example) => (
              <article key={example.title} className={styles.exampleCard}>
                <div className={styles.exampleHead}>
                  <span className={styles.exampleTag}>{example.label}</span>
                  <h3>{example.title}</h3>
                  <p>{example.description}</p>
                </div>
                <pre className={styles.exampleCode}>
                  <code>{example.code}</code>
                </pre>
              </article>
            ))}
          </div>
        </section>
      </main>
    </Layout>
  );
}
