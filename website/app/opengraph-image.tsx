import { ImageResponse } from "next/og";
import { SITE_NAME, SITE_TAGLINE } from "@/lib/site";
export const alt = `${SITE_NAME} — ${SITE_TAGLINE}`;
export const size = { width: 1200, height: 630 };
export const contentType = "image/png";
export const dynamic = "force-static";
export default function OpenGraphImage() {
  return new ImageResponse(
    (
      <div
        style={{
          display: "flex",
          width: "100%",
          height: "100%",
          background: "#f7f8f2",
          color: "#202727",
          flexDirection: "column",
        }}
      >
        <div
          style={{
            display: "flex",
            height: 88,
            alignItems: "center",
            justifyContent: "space-between",
            padding: "0 48px",
            borderBottom: "1px solid #d8ddd5",
          }}
        >
          <div style={{ display: "flex", alignItems: "center", gap: 13 }}>
            <svg width="33" height="33" viewBox="0 0 32 32" fill="#2357e9">
              <path d="M3 3h11v6H9v5h5v6H9v9H3V3Zm15 0h11v6H18V3Zm0 11h11v6H18v-6Zm0 9h11v6H18v-6Z" />
            </svg>
            <span style={{ fontSize: 27, letterSpacing: -1 }}>
              theboringfloor.
            </span>
          </div>
          <span style={{ fontSize: 14, color: "#586260" }}>
            OPEN SOURCE. OPEN POSSIBILITIES.
          </span>
        </div>
        <div style={{ display: "flex", flex: 1 }}>
          <div
            style={{
              display: "flex",
              flexDirection: "column",
              width: "69%",
              padding: "54px 48px",
              justifyContent: "space-between",
            }}
          >
            <div
              style={{
                display: "flex",
                flexDirection: "column",
                fontSize: 89,
                letterSpacing: -5,
                lineHeight: 1.04,
              }}
            >
              <span>Big ideas.</span>
              <span>Meet your team.</span>
            </div>
            <span
              style={{
                fontSize: 23,
                lineHeight: 1.45,
                maxWidth: 530,
                color: "#586260",
              }}
            >
              Your projects. Your agents. One happy place to work.
            </span>
            <span style={{ fontSize: 17, color: "#2357e9" }}>
              boringfloor.com →
            </span>
          </div>
          <div
            style={{
              display: "flex",
              position: "relative",
              width: "31%",
              background: "#2357e9",
              alignItems: "center",
              justifyContent: "center",
            }}
          >
            <svg width="330" height="400" viewBox="0 0 330 400">
              <path d="m35 263 130-74 130 74-130 74Z" fill="#123faf" />
              <path d="m35 237 130-74 130 74-130 74Z" fill="#8eb5ff" />
              <path d="m62 191 103-59 103 59-103 59Z" fill="#e3edff" />
              <path d="m62 191 103 59v36L62 227Z" fill="#548af0" />
              <path d="m165 250 103-59v36l-103 59Z" fill="#174abb" />
              <path d="m105 151 60-34 60 34-60 34Z" fill="#fffdf5" />
              <path d="m105 151 60 34v32l-60-34Z" fill="#6b9af4" />
              <path d="m165 185 60-34v32l-60 34Z" fill="#123faf" />
              <path d="M165 64v30m-15-15h30" stroke="#fffdf5" strokeWidth="5" />
            </svg>
          </div>
        </div>
        <div
          style={{
            display: "flex",
            height: 49,
            alignItems: "center",
            padding: "0 48px",
            fontSize: 12,
            borderTop: "1px solid #d8ddd5",
            color: "#586260",
          }}
        >
          THE OPEN-SOURCE TERMINAL OFFICE FOR OPENCODE, CLAUDE CODE, AND CODEX.
        </div>
      </div>
    ),
    { ...size },
  );
}
