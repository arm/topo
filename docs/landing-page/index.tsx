import Link from "@docusaurus/Link";
import useBaseUrl from "@docusaurus/useBaseUrl";
import Layout from "@theme/Layout";
import type { ReactElement } from "react";

import { homepageContent } from "./_homepage";
import styles from "./index.module.css";

function joinClasses(...parts: Array<string | undefined | false>): string {
  return parts.filter(Boolean).join(" ");
}

export default function Home(): ReactElement {
  const topoOverviewDiagramUrl = useBaseUrl("/img/topo-overview.svg");
  const tutorialCards = homepageContent.bottomCards.filter((card) =>
    (card.to ?? "").startsWith("/tutorials/"),
  );

  const otherCards = homepageContent.bottomCards.filter(
    (card) => !(card.to ?? "").startsWith("/tutorials/"),
  );

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
              {homepageContent.heroLinks.map((link) =>
                link.href ? (
                  <a
                    key={link.label}
                    href={link.href}
                    target="_blank"
                    rel="noreferrer"
                    className={joinClasses(
                      `button button--${link.variant}`,
                      styles.button,
                    )}
                  >
                    {link.label}
                    <svg
                      className={styles.externalLinkIcon}
                      viewBox="0 0 16 16"
                      aria-hidden="true"
                      focusable="false"
                    >
                      <path
                        d="M9.5 2.5h4v4M13.25 2.75 7.5 8.5M12.5 8v4.5h-9v-9H8"
                        fill="none"
                        stroke="currentColor"
                        strokeLinecap="round"
                        strokeLinejoin="round"
                        strokeWidth="1.5"
                      />
                    </svg>
                  </a>
                ) : (
                  <Link
                    key={link.label}
                    className={joinClasses(
                      `button button--${link.variant}`,
                      styles.button,
                    )}
                    to={link.to ?? "/"}
                  >
                    {link.label}
                  </Link>
                ),
              )}
            </div>
          </div>
          <div className={styles.heroVisual}>
            <img
              className={styles.heroDiagram}
              src={topoOverviewDiagramUrl}
              alt="Topo host-to-target deployment and development loop"
            />
          </div>
        </section>

        <section className={styles.bottom}>
          {otherCards.map((card) => {
            const cardContent = (
              <>
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
              </>
            );

            if (card.disabled) {
              return (
                <div key={card.title} className={styles.bottomCard}>
                  {cardContent}
                </div>
              );
            }

            if (card.href) {
              return (
                <a
                  key={card.title}
                  href={card.href}
                  target="_blank"
                  rel="noreferrer"
                  className={joinClasses(
                    styles.bottomCard,
                    styles.bottomCardLink,
                  )}
                >
                  {cardContent}
                </a>
              );
            }

            return (
              <Link
                key={card.title}
                to={card.to ?? "/tutorials/getting-started/your-first-program"}
                className={joinClasses(styles.bottomCard, styles.bottomCardLink)}
              >
                {cardContent}
              </Link>
            );
          })}

          {tutorialCards.length > 0 && (
            <div className={styles.tutorialsBox}>
              <div className={styles.tutorialsHeader}>
                <p className={styles.label}>Tutorials</p>
                <h4>Get started with guided walkthroughs</h4>
              </div>
              <div className={styles.tutorialsGrid}>
                {tutorialCards.map((card) => (
                  <div key={card.title} className={styles.bottomCard}>
                    <div>
                      <p className={styles.label}>{card.label}</p>
                      <h4>{card.title}</h4>
                      <p className={styles.small}>{card.description}</p>
                    </div>
                    {card.href ? (
                      <a
                        href={card.href}
                        target="_blank"
                        rel="noreferrer"
                        className={joinClasses(
                          "button button--sm",
                          `button--${card.variant}`,
                        )}
                      >
                        {card.cta}
                      </a>
                    ) : (
                      <Link
                        to={
                          card.to ??
                          "/tutorials/getting-started/your-first-program"
                        }
                        className={joinClasses(
                          "button button--sm",
                          `button--${card.variant}`,
                        )}
                      >
                        {card.cta}
                      </Link>
                    )}
                  </div>
                ))}
              </div>
            </div>
          )}
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
                <p className={styles.exampleLang}>{example.language}</p>
              </article>
            ))}
          </div>
        </section>
      </main>
    </Layout>
  );
}
