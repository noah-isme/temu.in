import HeroSection from '../components/landing/HeroSection';
import FeatureCards from '../components/landing/FeatureCards';
import StatsSection from '../components/landing/StatsSection';

export default function HomePage() {
  return (
    <div className="flex flex-col gap-12 py-4">
      <HeroSection />
      <FeatureCards />
      <StatsSection />
    </div>
  );
}
