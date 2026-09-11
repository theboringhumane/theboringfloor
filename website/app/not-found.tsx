import Link from "next/link";
import { ArrowUpRight } from "lucide-react";
import { SiteHeader } from "@/components/site-header";
import { SiteFooter } from "@/components/site-footer";
import { PageCover } from "@/components/page-cover";

export default function NotFound() {
  return (
    <div className="blue-site">
      <SiteHeader framed />
      <main id="main-content">
        <PageCover
          eyebrow="404 / A WRONG TURN ON THE FLOOR"
          title={
            <>
              This room is
              <br />
              <span className="blue-text">still a blueprint.</span>
            </>
          }
          description="We couldn’t find that page. Head back to the office, or let the manual point you in the right direction."
          art="plan"
        >
          <Link className="button-primary" href="/">
            Back to the office <ArrowUpRight size={18} />
          </Link>
          <Link className="text-link" href="/docs">
            Open the manual <ArrowUpRight size={17} />
          </Link>
        </PageCover>
      </main>
      <SiteFooter />
    </div>
  );
}
