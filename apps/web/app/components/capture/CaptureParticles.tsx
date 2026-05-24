const particles = [
  { left: "11%", top: "44%", size: 9 },
  { left: "36%", top: "23%", size: 9 },
  { left: "62%", top: "18%", size: 7 },
  { left: "85%", top: "22%", size: 9 },
  { left: "91%", top: "6%", size: 7 },
  { left: "83%", top: "29%", size: 9 },
  { left: "59%", top: "52%", size: 4 },
  { left: "57%", top: "76%", size: 5 },
];

export function CaptureParticles() {
  return (
    <div className="capture-particles" aria-hidden="true">
      {particles.map((particle) => (
        <span
          key={`${particle.left}-${particle.top}`}
          style={{
            left: particle.left,
            top: particle.top,
            width: `${particle.size}px`,
            height: `${particle.size}px`,
          }}
        />
      ))}
    </div>
  );
}
