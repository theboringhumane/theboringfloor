type ArtKind = "floor" | "plan" | "memory" | "network" | "source";

/** Original geometric artwork. SVG keeps the visual system crisp at every size. */
export function BlueprintArt({
  kind = "floor",
  className = "",
}: {
  kind?: ArtKind;
  className?: string;
}) {
  return (
    <svg
      viewBox="0 0 600 380"
      className={`blueprint-art ${className}`}
      fill="none"
      aria-hidden="true"
    >
      <path
        d="M0 76h600M0 152h600M0 228h600M0 304h600M100 0v380M200 0v380M300 0v380M400 0v380M500 0v380"
        stroke="currentColor"
        opacity=".1"
      />
      {kind === "floor" && (
        <g className="art-float">
          <path d="m95 252 205-116 205 116-205 115Z" fill="#123faf" />
          <path d="m95 230 205-116 205 116-205 115Z" fill="#6597fc" />
          <path d="m147 181 153-87 153 87-153 87Z" fill="#e8efff" />
          <path d="m147 181 153 87v43l-153-86Z" fill="#2d61dc" />
          <path d="m300 268 153-87v44l-153 86Z" fill="#174abf" />
          <path d="m193 155 107-61 107 61-107 61Z" fill="#b6d0ff" />
          <path d="m193 155 107 61v34l-107-60Z" fill="#487fec" />
          <path d="m300 216 107-61v35l-107 60Z" fill="#2962dd" />
          <path d="m257 110 43-25 43 25-43 24Z" fill="#fffdf5" />
          <path d="m257 110 43 24v36l-43-24Z" fill="#2c66ee" />
          <path d="m300 134 43-24v36l-43 24Z" fill="#174bbf" />
          <path d="M300 55v28m-12-16h24" stroke="#fffdf5" strokeWidth="3" />
          <path
            d="m194 241 33 19m-17-35 34 20m116-1 28-16m-12 33 30-17"
            stroke="#b6d0ff"
            strokeWidth="6"
          />
        </g>
      )}
      {kind === "plan" && (
        <g>
          <path
            d="M94 290h105v-75h106v-76h105V65h96"
            stroke="#6398ff"
            strokeWidth="34"
          />
          <path
            d="M94 273h105v-75h106v-76h105V48h96"
            stroke="#c5d9ff"
            strokeWidth="34"
          />
          <g fill="#fffdf5">
            <rect x="76" y="225" width="36" height="36" />
            <rect x="287" y="74" width="36" height="36" />
            <rect x="488" y="1" width="36" height="36" />
          </g>
          <path
            d="m82 242 8 8 17-17m188-142 8 8 17-17"
            stroke="#2464ed"
            strokeWidth="4"
          />
          <path
            d="M130 325h90m-40-15v30M348 280h120m-60-15v30"
            stroke="currentColor"
            opacity=".3"
          />
        </g>
      )}
      {kind === "memory" && (
        <g>
          {[0, 1, 2, 3].map((i) => (
            <g key={i} transform={`translate(0 ${-i * 48})`}>
              <path
                d="m158 280 142-80 142 80-142 80Z"
                fill={i === 3 ? "#e8efff" : "#91b7ff"}
              />
              <path d="m158 280 142 80v20l-142-80Z" fill="#3875f1" />
              <path d="m300 360 142-80v20l-142 80Z" fill="#1746b6" />
            </g>
          ))}
          <path
            d="M300 22v88"
            stroke="#fffdf5"
            strokeDasharray="4 7"
            strokeWidth="3"
          />
          <rect x="286" y="55" width="28" height="28" fill="#2464ed" />
        </g>
      )}
      {kind === "network" && (
        <g>
          <path
            d="M145 95h155v190h160M145 285h155V95h160M300 190h190"
            stroke="#86aeff"
            strokeWidth="3"
            strokeDasharray="8 8"
          />
          {[
            [145, 95],
            [145, 285],
            [460, 95],
            [460, 285],
            [300, 190],
          ].map(([x, y], i) => (
            <g key={i}>
              <rect
                x={x - 34}
                y={y - 34}
                width="68"
                height="68"
                fill={i === 4 ? "#fffdf5" : "#4785ff"}
              />
              <rect
                x={x - 10}
                y={y - 10}
                width="20"
                height="20"
                fill={i === 4 ? "#2464ed" : "#c8dcff"}
              />
            </g>
          ))}
        </g>
      )}
      {kind === "source" && (
        <g>
          <path
            d="m205 100-90 90 90 90m190-180 90 90-90 90m-61-220-68 260"
            stroke="#9bbeff"
            strokeWidth="24"
          />
          <path
            d="m205 88-90 90 90 90m190-180 90 90-90 90m-61-220-68 260"
            stroke="#fffdf5"
            strokeWidth="24"
          />
        </g>
      )}
      <g fill="currentColor" opacity=".5">
        <rect x="24" y="24" width="4" height="4" />
        <rect x="572" y="24" width="4" height="4" />
        <rect x="24" y="352" width="4" height="4" />
        <rect x="572" y="352" width="4" height="4" />
      </g>
    </svg>
  );
}
