import { BlueprintArt } from "@/components/home/blueprint-art";
export function PageCover({
  eyebrow,
  title,
  description,
  art = "floor",
  children,
}: {
  eyebrow: string;
  title: React.ReactNode;
  description: string;
  art?: "floor" | "plan" | "memory" | "network" | "source";
  children?: React.ReactNode;
}) {
  return (
    <section className="page-cover">
      <div className="page-cover-copy">
        <span className="eyebrow">{eyebrow}</span>
        <h1>{title}</h1>
        <p>{description}</p>
        {children && <div className="page-cover-actions">{children}</div>}
      </div>
      <div className={`page-cover-art page-cover-art--${art}`}>
        <BlueprintArt kind={art} />
        <span className="eyebrow">
          THEBORINGFLOOR / MAKE ROOM FOR POSSIBILITY
        </span>
      </div>
    </section>
  );
}
