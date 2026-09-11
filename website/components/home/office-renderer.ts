import * as THREE from "three";
import { mergeGeometries } from "three/addons/utils/BufferGeometryUtils.js";

type Controls = { paused: boolean; turn: number; update?: () => void };

/** A small, original low-poly office; no textures, model downloads, or post-processing. */
export function mountOffice(
  host: HTMLElement,
  controls: Controls,
  onReady: () => void,
  onLost: () => void,
) {
  let renderer: THREE.WebGLRenderer;
  try {
    renderer = new THREE.WebGLRenderer({
      alpha: true,
      antialias: true,
      powerPreference: "low-power",
    });
  } catch {
    return () => {};
  }
  const scene = new THREE.Scene();
  const camera = new THREE.OrthographicCamera(-13, 13, 8, -8, 0.1, 120);
  renderer.setPixelRatio(Math.min(window.devicePixelRatio, 1.5));
  renderer.outputColorSpace = THREE.SRGBColorSpace;
  renderer.setClearColor(0x000000, 0);
  host.appendChild(renderer.domElement);
  renderer.domElement.setAttribute("aria-hidden", "true");
  const ambient = new THREE.AmbientLight(0xffffff, 2.3);
  const sunlight = new THREE.DirectionalLight(0xffffff, 2.6);
  sunlight.position.set(-10, 22, 15);
  scene.add(ambient, sunlight);
  const office = new THREE.Group();
  scene.add(office);
  const materials = new Map<number, THREE.MeshLambertMaterial>();
  const geometries: THREE.BufferGeometry[] = [];
  const material = (color: number) => {
    if (!materials.has(color))
      materials.set(color, new THREE.MeshLambertMaterial({ color }));
    return materials.get(color)!;
  };
  const white = 0xdbe8ff,
    blue = 0x2560d8,
    ink = 0x103082,
    light = 0x739ff0,
    peach = 0xffd4af;
  function box(
    x: number,
    y: number,
    z: number,
    w: number,
    h: number,
    d: number,
    color: number,
    parent: THREE.Group = office,
  ) {
    const geometry = new THREE.BoxGeometry(w, h, d);
    geometries.push(geometry);
    const mesh = new THREE.Mesh(geometry, material(color));
    mesh.position.set(x, y, z);
    parent.add(mesh);
    return mesh;
  }
  // The whole floor sits on a deep cobalt plinth with a second floating step.
  box(0, -0.65, 0, 17.6, 0.3, 10.6, ink);
  box(0, -0.29, 0, 17, 0.4, 10, light);
  box(0, -0.02, 0, 16.7, 0.16, 9.7, white);
  for (let x = -8; x <= 8; x += 1)
    box(x, 0.069, 0, 0.012, 0.008, 9.7, 0x93b7ef);
  for (let z = -4; z <= 4; z += 1)
    box(0, 0.069, z, 16.7, 0.008, 0.012, 0x93b7ef);
  // Long gallery wall, repeated window bays, and a solid service core.
  box(0, 1.7, -4.7, 16.8, 3.4, 0.24, blue);
  for (let x = -7; x <= 3; x += 2.3) {
    box(x, 2.05, -4.53, 1.7, 2.3, 0.14, light);
    box(x, 2.05, -4.4, 0.06, 2.3, 0.1, white);
    box(x, 2.05, -4.4, 1.7, 0.06, 0.1, white);
    box(x, 0.88, -4.25, 1.95, 0.12, 0.55, white);
  }
  box(-8.2, 1.7, -2.7, 0.22, 3.4, 4.2, blue);
  box(-8.15, 3.4, -2.7, 0.3, 0.16, 4.35, white);
  box(0, 3.4, -4.7, 16.9, 0.16, 0.3, white);
  // Pinboard, made from individual blocks so it remains legible from above.
  box(5.9, 2.05, -4.47, 3, 1.75, 0.15, ink);
  for (let c = 0; c < 3; c++)
    for (let r = 0; r < 2; r++)
      box(
        5.04 + c * 0.85,
        2.5 - r * 0.7,
        -4.35,
        0.62,
        0.43,
        0.07,
        r === 0 ? white : light,
      );
  function desk(x: number, z: number, flipped = false) {
    const furniture = new THREE.Group();
    furniture.position.set(x, 0, z);
    if (flipped) furniture.rotation.y = Math.PI;
    office.add(furniture);
    box(0, 0.83, 0, 2.25, 0.15, 1.1, white, furniture);
    for (const dx of [-0.9, 0.9])
      box(dx, 0.43, 0, 0.12, 0.8, 0.85, blue, furniture);
    box(0, 1.32, -0.22, 0.9, 0.64, 0.12, ink, furniture);
    box(0, 1.32, -0.14, 0.73, 0.47, 0.03, 0x85c0ff, furniture);
    box(0, 1.01, -0.22, 0.1, 0.3, 0.1, blue, furniture);
    box(0, 0.94, 0.19, 0.7, 0.04, 0.24, light, furniture);
    box(0.81, 1.02, 0.15, 0.16, 0.23, 0.16, blue, furniture);
    box(0, 0.54, 1.07, 0.66, 0.16, 0.62, blue, furniture);
    box(0, 0.92, 1.33, 0.67, 0.75, 0.14, blue, furniture);
    box(0, 0.27, 1.07, 0.14, 0.45, 0.14, ink, furniture);
  }
  desk(-5, -2.6);
  desk(-1.8, -2.6);
  desk(1.4, -2.6);
  desk(-5, 2, true);
  desk(-1.8, 2, true);
  // Meeting table and four seats.
  box(4.75, 0.85, 1.5, 3.5, 0.18, 1.7, blue);
  box(4.75, 0.45, 1.5, 2.3, 0.8, 0.7, light);
  for (const x of [3.7, 5.6])
    for (const z of [0.05, 2.95]) {
      box(x, 0.52, z, 0.62, 0.18, 0.6, ink);
      box(x, 0.24, z, 0.12, 0.4, 0.12, blue);
    }
  box(4.8, 1, 1.5, 0.8, 0.1, 0.6, white);
  box(3.6, 1, 1.4, 0.35, 0.12, 0.3, peach);
  // Server rack and a tiny blue tea machine.
  box(7.2, 1.1, -2.75, 1.2, 2.2, 1.1, ink);
  for (let y = 0.4; y < 2.1; y += 0.4) {
    box(7.2, y, -2.16, 0.94, 0.22, 0.05, blue);
    box(7.53, y, -2.12, 0.07, 0.07, 0.03, 0x9dffcd);
  }
  box(-7.2, 0.67, -0.1, 1.1, 1.35, 1.1, light);
  box(-7.2, 1.58, -0.1, 0.8, 0.5, 0.75, white);
  box(-7.2, 1.58, 0.3, 0.48, 0.23, 0.05, ink);
  // Architectural plants: squared foliage for the pixel-inspired identity.
  function plant(x: number, z: number) {
    box(x, 0.27, z, 0.6, 0.54, 0.6, white);
    box(x, 0.8, z, 0.13, 0.8, 0.13, ink);
    box(x, 1.33, z, 0.86, 0.7, 0.7, blue);
    box(x - 0.24, 1.55, z + 0.08, 0.52, 0.52, 0.5, light);
    box(x + 0.22, 1.1, z + 0.12, 0.5, 0.6, 0.6, 0x2c77c2);
  }
  plant(-7, 3.6);
  plant(7.1, 3.6);
  plant(4.1, -3.6);
  // Coworkers each have a head, torso, arms, and independently animated legs.
  const workers: {
    group: THREE.Group;
    left: THREE.Mesh;
    right: THREE.Mesh;
    base: number;
    walking: boolean;
  }[] = [];
  function person(x: number, z: number, color: number, walking = false) {
    const group = new THREE.Group();
    group.position.set(x, 0.08, z);
    office.add(group);
    box(0, 0.92, 0, 0.36, 0.36, 0.35, peach, group);
    box(0, 1.1, -0.045, 0.39, 0.14, 0.37, ink, group);
    box(0, 0.57, 0, 0.43, 0.4, 0.27, color, group);
    box(-0.28, 0.55, 0, 0.13, 0.34, 0.17, color, group);
    box(0.28, 0.55, 0, 0.13, 0.34, 0.17, color, group);
    const left = box(-0.12, 0.22, 0, 0.16, 0.4, 0.18, ink, group);
    const right = box(0.12, 0.22, 0, 0.16, 0.4, 0.18, ink, group);
    workers.push({ group, left, right, base: x, walking });
  }
  person(-5, -1.45, blue);
  person(-1.8, -1.45, 0x5c91ec);
  person(1.4, -1.45, blue);
  person(-5, 0.85, blue);
  person(3.7, 2.8, white);
  person(5.6, 0.15, blue);
  person(-2, 3.8, blue, true);
  person(1, 0, 0x90b6ff, true);
  // A short staircase at the front gives the scene its inviting entrance.
  for (let i = 0; i < 3; i++)
    box(
      0.4,
      -0.65 + i * 0.19,
      5.6 - i * 0.35,
      2,
      0.2,
      0.75,
      i % 2 ? light : blue,
    );
  // Batch the stationary geometry by material. Only the two strolling agents
  // keep independent meshes, reducing the scene to a few dozen draw calls.
  office.updateMatrixWorld(true);
  const moving = new Set(
    workers.filter((worker) => worker.walking).map((worker) => worker.group),
  );
  const batches = new Map<THREE.Material, THREE.BufferGeometry[]>();
  const stationary: THREE.Object3D[] = [];
  for (const child of [...office.children]) {
    if (moving.has(child as THREE.Group)) continue;
    child.traverse((object) => {
      if (!(object instanceof THREE.Mesh)) return;
      const geometry = object.geometry.clone().applyMatrix4(object.matrixWorld);
      const mat = object.material as THREE.Material;
      if (!batches.has(mat)) batches.set(mat, []);
      batches.get(mat)!.push(geometry);
    });
    stationary.push(child);
  }
  stationary.forEach((object) => office.remove(object));
  batches.forEach((parts, mat) => {
    const merged = mergeGeometries(parts);
    parts.forEach((part) => part.dispose());
    if (merged) {
      geometries.push(merged);
      office.add(new THREE.Mesh(merged, mat));
    }
  });
  const pointer = { x: 0, y: 0 };
  let inView = true,
    frame = 0,
    last = 0,
    time = 0,
    turn = 0;
  const media = window.matchMedia("(prefers-reduced-motion: reduce)");
  let reduced = media.matches;
  function schedule() {
    if (!frame && inView && !document.hidden)
      frame = requestAnimationFrame(tick);
  }
  function renderFrame() {
    const angle = 0.64 + turn + (reduced ? 0 : pointer.x * 0.08);
    camera.position.set(
      Math.sin(angle) * 28,
      24 + (reduced ? 0 : pointer.y * 1.1),
      Math.cos(angle) * 28,
    );
    camera.lookAt(0, 0.2, 0);
    renderer.render(scene, camera);
  }
  const onMedia = () => {
    reduced = media.matches;
    renderFrame();
    schedule();
  };
  const onVisibility = () => {
    if (document.hidden) {
      cancelAnimationFrame(frame);
      frame = 0;
    } else schedule();
  };
  const onPointer = (event: PointerEvent) => {
    if (controls.paused || reduced) return;
    const rect = host.getBoundingClientRect();
    pointer.x = (event.clientX - rect.left) / rect.width - 0.5;
    pointer.y = (event.clientY - rect.top) / rect.height - 0.5;
    schedule();
  };
  const onLeave = () => {
    pointer.x = 0;
    pointer.y = 0;
  };
  function tick(now: number) {
    frame = 0;
    if (!inView || document.hidden) return;
    const turning = Math.abs(turn - controls.turn) > 0.001;
    if ((controls.paused || reduced) && !turning) return;
    schedule();
    if (now - last < 33) return;
    const delta = Math.min((now - last) / 1000, 0.1);
    last = now;
    turn = reduced
      ? controls.turn
      : THREE.MathUtils.lerp(turn, controls.turn, 0.075);
    if (!reduced && !controls.paused) {
      time += delta;
      for (const worker of workers)
        if (worker.walking) {
          worker.group.position.x =
            worker.base + Math.sin(time * 0.4 + worker.base) * 1.1;
          worker.group.rotation.y =
            Math.cos(time * 0.4 + worker.base) > 0 ? Math.PI / 2 : -Math.PI / 2;
          worker.left.rotation.x = Math.sin(time * 6) * 0.35;
          worker.right.rotation.x = -Math.sin(time * 6) * 0.35;
        }
      office.position.y = Math.sin(time * 0.65) * 0.045;
    }
    renderFrame();
  }
  controls.update = () => {
    if (reduced) {
      turn = controls.turn;
      renderFrame();
    } else schedule();
  };
  const resize = () => {
    const width = host.clientWidth,
      height = host.clientHeight;
    if (!width || !height) return;
    const aspect = width / height;
    const halfWidth = Math.max(11.8, 8.6 * aspect);
    camera.left = -halfWidth;
    camera.right = halfWidth;
    camera.top = halfWidth / aspect;
    camera.bottom = -halfWidth / aspect;
    camera.updateProjectionMatrix();
    renderer.setSize(width, height);
    renderFrame();
  };
  const observer = new ResizeObserver(resize);
  observer.observe(host);
  const intersection = new IntersectionObserver(([entry]) => {
    inView = entry.isIntersecting;
    if (inView) {
      renderFrame();
      schedule();
    } else {
      cancelAnimationFrame(frame);
      frame = 0;
    }
  });
  intersection.observe(host);
  const contextLost = (event: Event) => {
    event.preventDefault();
    cancelAnimationFrame(frame);
    onLost();
  };
  renderer.domElement.addEventListener("webglcontextlost", contextLost);
  host.addEventListener("pointermove", onPointer);
  host.addEventListener("pointerleave", onLeave);
  media.addEventListener("change", onMedia);
  document.addEventListener("visibilitychange", onVisibility);
  resize();
  onReady();
  schedule();
  return () => {
    cancelAnimationFrame(frame);
    observer.disconnect();
    intersection.disconnect();
    controls.update = undefined;
    document.removeEventListener("visibilitychange", onVisibility);
    host.removeEventListener("pointermove", onPointer);
    host.removeEventListener("pointerleave", onLeave);
    media.removeEventListener("change", onMedia);
    renderer.domElement.removeEventListener("webglcontextlost", contextLost);
    geometries.forEach((geometry) => geometry.dispose());
    materials.forEach((mat) => mat.dispose());
    renderer.dispose();
    renderer.domElement.remove();
  };
}
