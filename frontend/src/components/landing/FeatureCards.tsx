import { useRef, useState } from 'react';
import { motion, useMotionValue, useSpring, useTransform } from 'framer-motion';
import { FEATURES, Feature } from '../../shared/features';
import { Sparkles } from 'lucide-react';

function TiltCard({ feature, index }: { feature: Feature; index: number }) {
  const cardRef = useRef<HTMLDivElement>(null);

  // Mouse position state for spotlight glow & 3D tilt
  const mouseX = useMotionValue(0);
  const mouseY = useMotionValue(0);

  const rotateX = useSpring(useTransform(mouseY, [-0.5, 0.5], [10, -10]), { stiffness: 300, damping: 25 });
  const rotateY = useSpring(useTransform(mouseX, [-0.5, 0.5], [-10, 10]), { stiffness: 300, damping: 25 });

  const [glowPos, setGlowPos] = useState({ x: 50, y: 50 });
  const [isHovered, setIsHovered] = useState(false);

  const handleMouseMove = (e: React.MouseEvent<HTMLDivElement>) => {
    if (!cardRef.current) return;
    const rect = cardRef.current.getBoundingClientRect();
    const width = rect.width;
    const height = rect.height;

    const xPct = (e.clientX - rect.left) / width - 0.5;
    const yPct = (e.clientY - rect.top) / height - 0.5;

    mouseX.set(xPct);
    mouseY.set(yPct);

    setGlowPos({
      x: ((e.clientX - rect.left) / width) * 100,
      y: ((e.clientY - rect.top) / height) * 100,
    });
  };

  const handleMouseLeave = () => {
    mouseX.set(0);
    mouseY.set(0);
    setIsHovered(false);
  };

  const IconComponent = feature.icon;

  return (
    <motion.div
      initial={{ opacity: 0, y: 30 }}
      whileInView={{ opacity: 1, y: 0 }}
      viewport={{ once: true, margin: '-50px' }}
      transition={{ duration: 0.5, delay: index * 0.1, type: 'spring', stiffness: 200 }}
      style={{
        perspective: 1000,
      }}
    >
      <motion.div
        ref={cardRef}
        onMouseMove={handleMouseMove}
        onMouseEnter={() => setIsHovered(true)}
        onMouseLeave={handleMouseLeave}
        style={{
          rotateX,
          rotateY,
          transformStyle: 'preserve-3d',
        }}
        className="group relative overflow-hidden rounded-2xl border border-slate-200/80 bg-white p-6 shadow-sm transition-shadow duration-300 hover:shadow-xl hover:shadow-primary/10"
      >
        {/* Dynamic Cursor Spotlight Effect */}
        {isHovered && (
          <div
            className="pointer-events-none absolute -inset-px transition-opacity duration-300"
            style={{
              background: `radial-gradient(400px circle at ${glowPos.x}% ${glowPos.y}%, rgba(99, 102, 241, 0.12), transparent 80%)`,
            }}
          />
        )}

        <div className="relative z-10 flex flex-col h-full justify-between">
          <div>
            <div className="mb-4 inline-flex h-14 w-14 items-center justify-center rounded-2xl bg-primary/10 text-primary transition-transform duration-300 group-hover:scale-110 group-hover:rotate-6">
              <IconComponent className="h-7 w-7 transition-transform duration-300 group-hover:scale-110" />
            </div>

            <h3 className="mb-2 text-xl font-bold text-slate-900 group-hover:text-primary transition-colors">
              {feature.title}
            </h3>

            <p className="text-sm leading-relaxed text-slate-600">
              {feature.description}
            </p>
          </div>

          <div className="mt-6 flex items-center gap-1.5 text-xs font-semibold text-primary opacity-0 transition-all duration-300 group-hover:opacity-100 group-hover:translate-x-1">
            <Sparkles className="h-3.5 w-3.5" />
            <span>Interactive Module</span>
          </div>
        </div>
      </motion.div>
    </motion.div>
  );
}

export default function FeatureCards() {
  return (
    <section className="mt-16">
      <div className="mb-8 text-center">
        <span className="text-xs font-bold uppercase tracking-widest text-primary">Fitur Unggulan</span>
        <h2 className="mt-2 text-3xl font-extrabold text-slate-900 sm:text-4xl">
          Arsitektur Booking Modern & Skalabel
        </h2>
        <p className="mt-3 text-slate-600">
          Setiap modul dirancang dengan responsivitas tinggi dan integrasi mulus.
        </p>
      </div>

      <div className="grid gap-6 sm:grid-cols-2">
        {FEATURES.map((feature, idx) => (
          <TiltCard key={feature.title} feature={feature} index={idx} />
        ))}
      </div>
    </section>
  );
}
