"use client";

import { ProductScreenshot } from "@/components/product-screenshot";
import { useId, useState } from "react";
import Link from "next/link";
import {
  ArrowUpRight,
  FileCode2,
  Layers3,
  ListTodo,
  MessagesSquare,
} from "lucide-react";
const views = [
  {
    title: "The floor",
    icon: Layers3,
    shot: "transcript",
    heading: "See the team. Follow the work.",
    text: "A living office on the left. The full conversation on the right. Every coworker, tool call, and handoff stays in view.",
    href: "/docs/workspaces",
    link: "Explore workspaces",
  },
  {
    title: "The board",
    icon: ListTodo,
    shot: "board",
    heading: "Good work has a place to land.",
    text: "Turn ideas into persistent tickets. Keep acceptance criteria, linked conversations, and results together as work moves across the board.",
    href: "/docs/queue-board-memory",
    link: "Meet the board",
  },
  {
    title: "The conversation",
    icon: MessagesSquare,
    shot: "expanded-transcript",
    heading: "Get the whole conversation.",
    text: "Open a work thread to see what an agent is doing. Read the tools, the reasoning, and the result without leaving your project.",
    href: "/docs/chat-and-threads",
    link: "Follow work threads",
  },
  {
    title: "The code",
    icon: FileCode2,
    shot: "files",
    heading: "Keep your code within reach.",
    text: "Browse project files beside the conversation. Your terminal, git changes, and project context are part of the same workspace.",
    href: "/docs/terminal-and-git-tabs",
    link: "Explore the tools",
  },
];
export function ProductExplorer() {
  const [selected, setSelected] = useState(0);
  const id = useId();
  const view = views[selected];
  return (
    <div className="product-explorer">
      <div
        className="explorer-tabs"
        role="tablist"
        aria-label="Explore the workspace"
      >
        {views.map((item, i) => (
          <button
            key={item.title}
            role="tab"
            id={`${id}-tab-${i}`}
            aria-selected={i === selected}
            aria-controls={`${id}-panel`}
            tabIndex={i === selected ? 0 : -1}
            onClick={() => setSelected(i)}
            onKeyDown={(event) => {
              let next = i;
              if (event.key === "ArrowRight") next = (i + 1) % views.length;
              else if (event.key === "ArrowLeft")
                next = (i + views.length - 1) % views.length;
              else if (event.key === "Home") next = 0;
              else if (event.key === "End") next = views.length - 1;
              else return;
              event.preventDefault();
              setSelected(next);
              document.getElementById(`${id}-tab-${next}`)?.focus();
            }}
          >
            <item.icon size={17} />
            {item.title}
            <span>0{i + 1}</span>
          </button>
        ))}
      </div>
      <div
        id={`${id}-panel`}
        role="tabpanel"
        aria-labelledby={`${id}-tab-${selected}`}
        tabIndex={0}
        className="explorer-panel"
      >
        <div className="explorer-description" key={view.title}>
          <div>
            <h3>{view.heading}</h3>
            <p>{view.text}</p>
          </div>
          <Link href={view.href} className="text-link">
            {view.link}
            <ArrowUpRight size={17} />
          </Link>
        </div>
        <div className="product-screen">
          <div className="screen-chrome">
            <span>
              <i />
              <i />
              <i />
            </span>
            <span>theboringfloor — GitHub Light</span>
            <a
              href={`/shots/workspaces/${view.shot}.webp`}
              target="_blank"
              rel="noreferrer"
              aria-label={`Open full-size image: ${view.title}`}
            >
              View full size ↗
            </a>
          </div>
          <ProductScreenshot
            key={view.shot}
            src={`/shots/workspaces/${view.shot}.webp`}
            alt={view.heading + " Actual theboringfloor terminal interface."}
            width={1548}
            height={1014}
            loading="lazy"
          />
        </div>
      </div>
      <div className="explorer-caption">
        <span>ACTUAL INTERFACE / ILLUSTRATIVE PROJECT DATA</span>
        <span>macOS / Linux / Windows</span>
      </div>
    </div>
  );
}
