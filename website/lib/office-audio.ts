/** An original, quiet office soundscape. No recordings or network requests. */
export function createOfficeAudio() {
  const context = new AudioContext({ latencyHint: "playback" });
  const master = context.createGain();
  master.gain.value = 0;
  master.connect(context.destination);
  let active = false;
  let disposed = false;
  let volume = 0.18;
  let revision = 0;
  let suspendTimer: ReturnType<typeof setTimeout> | undefined;
  const timers = new Set<ReturnType<typeof setTimeout>>();
  const voices = new Set<() => void>();

  // Stereo brown noise gives the room a soft ventilation bed, without hiss.
  const roomBuffer = context.createBuffer(2, context.sampleRate * 8, context.sampleRate);
  const clickBuffer = context.createBuffer(1, context.sampleRate * 3, context.sampleRate);
  for (let channel = 0; channel < 2; channel++) {
    const data = roomBuffer.getChannelData(channel);
    let previous = 0;
    for (let i = 0; i < data.length; i++) {
      previous = (previous + (Math.random() * 2 - 1) * 0.02) / 1.02;
      data[i] = previous * 3.5;
    }
    // Close the loop smoothly without a click every eight seconds.
    const fade = Math.floor(context.sampleRate * 0.1);
    for (let i = 0; i < fade; i++) {
      const mix = i / (fade - 1);
      data[data.length - fade + i] = data[data.length - fade + i] * (1 - mix) + data[0] * mix;
    }
  }
  const noise = clickBuffer.getChannelData(0);
  for (let i = 0; i < noise.length; i++) noise[i] = Math.random() * 2 - 1;
  const room = context.createBufferSource();
  room.buffer = roomBuffer;
  room.loop = true;
  const roomFilter = context.createBiquadFilter();
  roomFilter.type = "lowpass";
  roomFilter.frequency.value = 650;
  const roomGain = context.createGain();
  roomGain.gain.value = 0.35;
  room.connect(roomFilter).connect(roomGain).connect(master);
  room.start();

  function voice(source: AudioScheduledSourceNode, duration: number, gain: number, frequency: number, pan: number, delay = 0, attack = 0.006) {
    const now = context.currentTime + delay;
    const filter = context.createBiquadFilter();
    filter.type = "bandpass";
    filter.frequency.value = frequency;
    filter.Q.value = 0.65;
    const envelope = context.createGain();
    envelope.gain.setValueAtTime(0, now);
    envelope.gain.linearRampToValueAtTime(gain, now + attack);
    envelope.gain.exponentialRampToValueAtTime(0.0001, now + duration);
    const panner = context.createStereoPanner();
    panner.pan.value = pan;
    source.connect(filter).connect(envelope).connect(panner).connect(master);
    const cleanup = () => {
      source.onended = null;
      try { source.stop(); } catch { /* Already finished. */ }
      source.disconnect(); filter.disconnect(); envelope.disconnect(); panner.disconnect();
      voices.delete(cleanup);
    };
    voices.add(cleanup);
    source.onended = cleanup;
    source.start(now);
    source.stop(now + duration + 0.02);
  }

  function rustle(duration: number, gain: number, frequency: number, pan: number, delay = 0, attack = 0.006) {
    const source = context.createBufferSource();
    source.buffer = clickBuffer;
    source.loop = true;
    voice(source, duration, gain, frequency, pan, delay, attack);
  }

  function motor(duration: number, gain: number, frequency: number, pan: number, delay = 0) {
    const oscillator = context.createOscillator();
    oscillator.type = "triangle";
    oscillator.frequency.setValueAtTime(frequency, context.currentTime + delay);
    oscillator.frequency.linearRampToValueAtTime(frequency * 0.82, context.currentTime + delay + duration);
    voice(oscillator, duration, gain, frequency * 2, pan, delay, 0.08);
  }

  function keyboard() {
    const pan = Math.random() * 1.1 - 0.55;
    let delay = 0;
    for (let i = 0; i < 4 + Math.floor(Math.random() * 6); i++) {
      delay += 0.075 + Math.random() * 0.17;
      rustle(0.024, 0.075 + Math.random() * 0.035, 1700 + Math.random() * 1400, pan, delay);
      rustle(0.045, 0.11, 340 + Math.random() * 100, pan, delay + 0.007);
    }
  }

  function printer() {
    motor(2.1, 0.13, 135, 0.45);
    for (let i = 0; i < 4; i++) rustle(0.32, 0.09, 850, 0.45, i * 0.42, 0.04);
    rustle(0.7, 0.055, 1600, 0.45, 1.9, 0.08);
  }

  function coffee() {
    motor(3.8, 0.13, 92, -0.5);
    rustle(4.2, 0.085, 1050, -0.5, 0.2, 0.6);
    // A few soft drips as the machine finishes.
    for (let i = 0; i < 5; i++) rustle(0.06, 0.08, 520, -0.5, 3.4 + i * 0.18);
  }

  function schedule(sound: () => void, delay: number, repeatMin: number, repeatRange: number) {
    const timer = setTimeout(() => {
      timers.delete(timer);
      if (!active || disposed) return;
      sound();
      schedule(sound, repeatMin + Math.random() * repeatRange, repeatMin, repeatRange);
    }, delay);
    timers.add(timer);
  }

  function clearSchedule() {
    timers.forEach(clearTimeout);
    timers.clear();
  }

  return {
    async setActive(next: boolean) {
      if (disposed || active === next) return;
      active = next;
      const current = ++revision;
      clearTimeout(suspendTimer);
      clearSchedule();
      if (next) {
        await context.resume();
        if (disposed || current !== revision) return;
        master.gain.cancelScheduledValues(context.currentTime);
        master.gain.setTargetAtTime(volume * 0.65, context.currentTime, 0.3);
        schedule(keyboard, 900, 2600, 3800);
        schedule(printer, 7000, 23000, 16000);
        schedule(coffee, 15000, 41000, 24000);
      } else {
        master.gain.cancelScheduledValues(context.currentTime);
        master.gain.setTargetAtTime(0, context.currentTime, 0.07);
        suspendTimer = setTimeout(() => {
          voices.forEach((cleanup) => cleanup());
          void context.suspend().catch(() => {});
        }, 350);
      }
    },
    setVolume(next: number) {
      volume = Math.min(0.5, Math.max(0, next));
      if (active && !disposed) master.gain.setTargetAtTime(volume * 0.65, context.currentTime, 0.12);
    },
    dispose() {
      disposed = true;
      active = false;
      revision++;
      clearTimeout(suspendTimer);
      clearSchedule();
      voices.forEach((cleanup) => cleanup());
      room.stop(); room.disconnect(); roomFilter.disconnect(); roomGain.disconnect(); master.disconnect();
      void context.close().catch(() => {});
    },
  };
}

export type OfficeAudio = ReturnType<typeof createOfficeAudio>;
