/**
 * データ取得中のローディング表示（スケルトン or スピナー）。
 * 白画面ではなく「読み込み中」が伝わる UX 用。
 */

export type SkeletonVariant = 'projectList' | 'projectDetail' | 'mePage' | 'spinner';

interface Props {
  variant: SkeletonVariant;
}

function ProjectListSkeleton() {
  return (
    <div className="project-list" style={{ marginTop: '2rem', display: 'flex', flexDirection: 'column', gap: '1rem' }}>
      {[1, 2, 3, 4].map((i) => (
        <div key={i} className="card skeleton skeleton-card" style={{ padding: '1rem' }}>
          <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'flex-start', marginBottom: '0.5rem' }}>
            <div style={{ flex: 1 }}>
              <div className="skeleton skeleton-title" style={{ marginBottom: '0.35rem' }} />
              <div className="skeleton skeleton-text" style={{ width: '40%', height: '0.8rem' }} />
            </div>
            <div className="skeleton" style={{ width: '3rem', height: '1.5rem', borderRadius: '4px' }} />
          </div>
          <div className="skeleton skeleton-line" style={{ marginBottom: '0.25rem' }} />
          <div className="skeleton skeleton-line" style={{ width: '85%', height: '0.8rem' }} />
        </div>
      ))}
    </div>
  );
}

function ProjectDetailSkeleton() {
  return (
    <div style={{ marginTop: '1rem' }}>
      <div className="skeleton skeleton-title" style={{ height: '1.75rem', width: '60%', marginBottom: '0.5rem' }} />
      <div className="skeleton skeleton-text" style={{ width: '30%', marginBottom: '1.5rem' }} />
      <div style={{ display: 'flex', gap: '0.5rem', marginBottom: '1.5rem' }}>
        <div className="skeleton skeleton-tab" />
        <div className="skeleton skeleton-tab" />
        <div className="skeleton skeleton-tab" />
      </div>
      <div className="skeleton skeleton-block" style={{ marginBottom: '1rem' }} />
      <div className="skeleton skeleton-line" style={{ marginBottom: '0.5rem' }} />
      <div className="skeleton skeleton-line" style={{ width: '90%' }} />
    </div>
  );
}

function MePageSkeleton() {
  return (
    <div style={{ marginTop: '1rem' }}>
      <div style={{ display: 'flex', gap: '0.5rem', marginBottom: '1.5rem' }}>
        <div className="skeleton skeleton-tab" />
        <div className="skeleton skeleton-tab" />
      </div>
      <div className="skeleton skeleton-block" style={{ minHeight: '12rem' }} />
    </div>
  );
}

function SpinnerFallback() {
  return (
    <div style={{ marginTop: '2rem', display: 'flex', alignItems: 'center', gap: '0.75rem', color: 'var(--color-text-muted)' }}>
      <span className="skeleton-spinner" aria-hidden />
      <span>読み込み中...</span>
    </div>
  );
}

export default function LoadingSkeleton({ variant }: Props) {
  switch (variant) {
    case 'projectList':
      return <ProjectListSkeleton />;
    case 'projectDetail':
      return <ProjectDetailSkeleton />;
    case 'mePage':
      return <MePageSkeleton />;
    case 'spinner':
    default:
      return <SpinnerFallback />;
  }
}
