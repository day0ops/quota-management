import { useState } from 'react';
import { Tooltip } from '../../components/common/Tooltip';
import styled from '@emotion/styled';
import { spacing, colors, fontSize, radius } from '../../styles';
import { BudgetForecast, TimeSeriesPoint } from '../../api/types';

// ── Styled components ─────────────────────────────────────────────────────────

const Section = styled.div`
  margin-bottom: ${spacing[6]};
  animation: fg-fade-up 0.4s ease-out both;
`;

const SectionHeader = styled.div`
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: ${spacing[4]};
`;

const SectionTitle = styled.h2`
  font-size: ${fontSize.lg};
  font-weight: 500;
  color: ${colors.foreground};
`;

const ConfidenceBadge = styled.span<{ $c: number }>`
  font-size: ${fontSize.xs};
  padding: ${spacing[1]} ${spacing[2]};
  border-radius: ${radius.full};
  background: ${({ $c }) =>
    $c >= 0.7
      ? 'rgba(34,197,94,0.12)'
      : $c >= 0.4
        ? 'rgba(251,191,36,0.12)'
        : 'rgba(148,163,184,0.12)'};
  color: ${({ $c }) =>
    $c >= 0.7
      ? 'var(--color-success)'
      : $c >= 0.4
        ? 'var(--color-warning)'
        : colors.mutedForeground};
`;

const StatsGrid = styled.div`
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(160px, 1fr));
  gap: ${spacing[4]};
  margin-bottom: ${spacing[5]};
`;

const StatCard = styled.div`
  background:
    linear-gradient(var(--color-card-bg), var(--color-card-bg)) padding-box,
    linear-gradient(135deg, rgba(104, 68, 255, 0.18) 0%, rgba(43, 34, 57, 0.06) 100%) border-box;
  border: 1px solid transparent;
  border-radius: 16px;
  padding: ${spacing[4]};
  box-shadow: 0 3.364px 3.364px 0 rgba(0, 0, 0, 0.25);
`;

const StatLabel = styled.div`
  font-size: ${fontSize.xs};
  color: ${colors.mutedForeground};
  margin-bottom: ${spacing[1]};
  text-transform: uppercase;
  letter-spacing: 0.04em;
`;

const StatValue = styled.div`
  font-size: ${fontSize.xl};
  font-weight: 600;
  color: ${colors.foreground};
`;

const StatSub = styled.div`
  font-size: ${fontSize.xs};
  color: ${colors.mutedForeground};
  margin-top: 2px;
`;

const ChartCard = styled.div`
  background:
    linear-gradient(var(--color-card-bg), var(--color-card-bg)) padding-box,
    linear-gradient(135deg, rgba(104, 68, 255, 0.18) 0%, rgba(43, 34, 57, 0.06) 100%) border-box;
  border: 1px solid transparent;
  border-radius: 16px;
  padding: ${spacing[5]};
  box-shadow: 0 3.364px 3.364px 0 rgba(0, 0, 0, 0.25);
`;

const ChartLegend = styled.div`
  display: flex;
  gap: ${spacing[5]};
  margin-bottom: ${spacing[4]};
`;

const LegendItem = styled.div`
  display: flex;
  align-items: center;
  gap: ${spacing[2]};
  font-size: ${fontSize.xs};
  color: ${colors.mutedForeground};
`;

const LegendDash = styled.div<{ color: string; dashed?: boolean; $opacity?: number }>`
  width: 20px;
  height: 0;
  border-top: 2px ${({ dashed }) => (dashed ? 'dashed' : 'solid')} ${({ color }) => color};
  opacity: ${({ $opacity }) => $opacity ?? 1};
`;

// ── Animated SVG elements ─────────────────────────────────────────────────────

const AnimPath = styled.path<{ $dur: string; $delay: string; $dasharray?: string }>`
  stroke-dasharray: ${({ $dasharray }) => $dasharray ?? '1'};
  stroke-dashoffset: 1;
  pathlength: 1;
  animation: fg-draw ${({ $dur }) => $dur} cubic-bezier(0.4, 0, 0.2, 1) ${({ $delay }) => $delay}
    forwards;
`;

const AnimProjPath = styled.path<{ $dur: string; $delay: string }>`
  stroke-dasharray: 6 4;
  opacity: 0;
  animation: fg-fade-up ${({ $dur }) => $dur} ease-out ${({ $delay }) => $delay} forwards;
`;

const AnimArea = styled.path<{ $delay: string }>`
  opacity: 0;
  animation: fg-fade-up 0.6s ease-out ${({ $delay }) => $delay} forwards;
`;

const AnimDot = styled.circle<{ $delay: string }>`
  opacity: 0;
  transform-box: fill-box;
  transform-origin: center;
  animation: fg-pop-in 0.25s ease-out ${({ $delay }) => $delay} both;
`;

const PulseDot = styled.circle<{ $delay: string }>`
  transform-box: fill-box;
  transform-origin: center;
  animation: fg-pulse 2.2s ease-in-out ${({ $delay }) => $delay} infinite;
`;

const TodayDot = styled.circle<{ $delay: string }>`
  opacity: 0;
  animation: fg-fade-up 0.3s ease-out ${({ $delay }) => $delay} forwards;
`;

// ── Helpers ───────────────────────────────────────────────────────────────────

function fmt(v: number): string {
  if (v === 0) return '$0.00';
  if (v >= 1000) return `$${(v / 1000).toFixed(1)}K`;
  if (v >= 1) return `$${v.toFixed(2)}`;
  if (v >= 0.01) return `$${v.toFixed(4)}`;
  return `$${v.toPrecision(2)}`;
}

function fmtShort(v: number): string {
  if (v === 0) return '$0';
  if (v >= 1000) return `$${(v / 1000).toFixed(1)}K`;
  if (v >= 1) return `$${v.toFixed(2)}`;
  return `$${v.toFixed(3)}`;
}

// ── Chart ─────────────────────────────────────────────────────────────────────

interface ChartPt {
  x: number;
  y: number;
  cum: number;
  date: string;
}
interface HoverState {
  x: number;
  y: number;
  date: string;
  value: number;
  type: 'actual' | 'projected';
}

function ForecastChart({ forecast }: { forecast: BudgetForecast }) {
  const [hover, setHover] = useState<HoverState | null>(null);

  const W = 640,
    H = 200;
  const ML = 46,
    MR = 16,
    MT = 14,
    MB = 28;
  const cW = W - ML - MR;
  const cH = H - MT - MB;

  const pts: TimeSeriesPoint[] = forecast.time_series || [];
  const n = pts.length;
  if (n < 2) return null;

  const boundaries = new Set(forecast.period_boundaries || []);
  const maxY = Math.max(forecast.budget_amount_usd, forecast.projected_spend_usd, 0.001) * 1.18;
  const xS = (i: number) => (i / Math.max(n - 1, 1)) * cW;
  const yS = (v: number) => cH - Math.min((v / maxY) * cH, cH);

  // Per-period cumulative actual: resets at each period boundary
  const nowMs = Date.now();
  const isActualPoint = (dateStr: string) => new Date(dateStr).getTime() <= nowMs;
  const isDatetime = n > 0 && pts[0].date.includes('T');

  const actualSegments: ChartPt[][] = [];
  let currentSeg: ChartPt[] = [];
  let segCum = 0;
  let lastActualIdx = -1;

  for (let i = 0; i < n; i++) {
    // Reset cumulative at period boundaries
    if (i > 0 && boundaries.has(pts[i].date)) {
      if (currentSeg.length > 0) actualSegments.push(currentSeg);
      currentSeg = [];
      segCum = 0;
    }
    if (isActualPoint(pts[i].date)) {
      segCum += pts[i].actual_usd ?? 0;
      currentSeg.push({ x: xS(i), y: yS(segCum), cum: segCum, date: pts[i].date });
      lastActualIdx = i;
    }
  }
  if (currentSeg.length > 0) actualSegments.push(currentSeg);
  const actualPts = actualSegments.flat();

  // Projected — continuing from current period's cumulative
  const projPts: ChartPt[] = [];
  if (lastActualIdx >= 0) {
    projPts.push({
      x: xS(lastActualIdx),
      y: yS(segCum),
      cum: segCum,
      date: pts[lastActualIdx].date,
    });
    let pc = segCum;
    for (let i = lastActualIdx + 1; i < n; i++) {
      pc += pts[i].projected_usd ?? 0;
      projPts.push({ x: xS(i), y: yS(pc), cum: pc, date: pts[i].date });
    }
  } else {
    let pc = 0;
    for (let i = 0; i < n; i++) {
      pc += pts[i].projected_usd ?? 0;
      projPts.push({ x: xS(i), y: yS(pc), cum: pc, date: pts[i].date });
    }
  }

  const svgLine = (arr: ChartPt[]) =>
    arr.length === 0
      ? ''
      : `M${arr[0].x},${arr[0].y} ` +
        arr
          .slice(1)
          .map(p => `L${p.x},${p.y}`)
          .join(' ');

  const svgArea = (arr: ChartPt[]) => {
    if (arr.length === 0) return '';
    const last = arr[arr.length - 1];
    return `${svgLine(arr)} L${last.x},${cH} L${arr[0].x},${cH} Z`;
  };

  const budgetY = yS(forecast.budget_amount_usd);
  const todayPt = actualPts[actualPts.length - 1];
  const lineDur = 1.5;

  // Labels — spread across full time range, use time labels for intra-day series
  const firstDay = pts[0] ? new Date(pts[0].date).toLocaleDateString() : '';
  const lastDay = pts[n - 1] ? new Date(pts[n - 1].date).toLocaleDateString() : '';
  const isIntraDay = isDatetime && firstDay === lastDay;
  const step = Math.max(1, Math.floor(n / 7));
  const xLabels = Array.from({ length: Math.ceil(n / step) }, (_, k) => {
    const i = k * step;
    const d = new Date(pts[i].date);
    let label: string;
    if (isIntraDay) {
      label = d.toLocaleTimeString(undefined, { hour: '2-digit', minute: '2-digit' });
    } else if (isDatetime) {
      label =
        d.toLocaleDateString(undefined, { month: 'short', day: 'numeric' }) +
        ' ' +
        d.toLocaleTimeString(undefined, { hour: '2-digit', minute: '2-digit' });
    } else {
      label = d.toLocaleDateString(undefined, { month: 'short', day: 'numeric' });
    }
    return { x: xS(i), label };
  });
  const yTicks = [0, 0.25, 0.5, 0.75, 1].map(t => ({ y: yS(maxY * t), label: fmtShort(maxY * t) }));

  // Boundary x-positions for reset markers
  const boundaryXs: { x: number; date: string }[] = [];
  for (let i = 0; i < n; i++) {
    if (boundaries.has(pts[i].date)) {
      boundaryXs.push({ x: xS(i), date: pts[i].date });
    }
  }

  // Mouse tracking
  const handleMouseMove = (e: React.MouseEvent<SVGRectElement>) => {
    const svgEl = (e.currentTarget as SVGElement).closest('svg')!;
    const r = svgEl.getBoundingClientRect();
    const rawX = ((e.clientX - r.left) / r.width) * W - ML;
    const svgX = Math.max(0, Math.min(cW, rawX));
    const idx = Math.round((svgX / cW) * (n - 1));
    const ap =
      actualPts[
        actualPts.findIndex((_, i) => {
          const ai = Math.round((actualPts[i].x / cW) * (n - 1));
          return ai === idx;
        })
      ];
    const pp =
      projPts[
        projPts.findIndex((_, i) => {
          const pi = Math.round((projPts[i].x / cW) * (n - 1));
          return pi === idx;
        })
      ];
    const pt = ap ?? pp;
    if (!pt) {
      setHover(null);
      return;
    }
    setHover({
      x: pt.x,
      y: pt.y,
      date: new Date(pt.date).toLocaleDateString(undefined, { month: 'short', day: 'numeric' }),
      value: pt.cum,
      type: ap ? 'actual' : 'projected',
    });
  };

  const accent = 'var(--color-chart-accent)';
  const actual = 'var(--color-chart-actual)';
  const gridC = 'rgba(128,128,128,0.07)';
  const labelC = 'var(--color-muted-foreground)';

  // Tooltip position
  const ttX = hover ? (hover.x > cW * 0.68 ? hover.x - 100 : hover.x + 10) : 0;
  const ttY = hover ? Math.max(0, hover.y - 40) : 0;

  return (
    <svg
      viewBox={`0 0 ${W} ${H}`}
      style={{ width: '100%', height: 'auto', display: 'block', overflow: 'visible' }}
    >
      <defs>
        <linearGradient id="fg-stroke-grad" x1="0%" y1="0%" x2="100%" y2="0%">
          <stop offset="0%" stopColor={accent} stopOpacity="0.65" />
          <stop offset="100%" stopColor={actual} />
        </linearGradient>
        <linearGradient id="fg-fill-actual" x1="0%" y1="0%" x2="0%" y2="100%">
          <stop offset="0%" stopColor={accent} stopOpacity="0.22" />
          <stop offset="100%" stopColor={accent} stopOpacity="0.01" />
        </linearGradient>
        <linearGradient id="fg-fill-proj" x1="0%" y1="0%" x2="0%" y2="100%">
          <stop offset="0%" stopColor={accent} stopOpacity="0.09" />
          <stop offset="100%" stopColor={accent} stopOpacity="0" />
        </linearGradient>
        <filter id="fg-glow" x="-50%" y="-50%" width="200%" height="200%">
          <feGaussianBlur stdDeviation="3" result="blur" />
          <feMerge>
            <feMergeNode in="blur" />
            <feMergeNode in="SourceGraphic" />
          </feMerge>
        </filter>
      </defs>

      <g transform={`translate(${ML},${MT})`}>
        {/* Grid */}
        {yTicks.map((t, i) => (
          <line key={i} x1={0} y1={t.y} x2={cW} y2={t.y} stroke={gridC} strokeWidth="1" />
        ))}

        {/* Budget limit */}
        <line
          x1={0}
          y1={budgetY}
          x2={cW}
          y2={budgetY}
          stroke="var(--color-chart-error)"
          strokeWidth="1"
          strokeDasharray="5 4"
          opacity="0.5"
        />
        <text
          x={cW - 2}
          y={budgetY - 4}
          fontSize="5.5"
          fill="var(--color-chart-error)"
          opacity="0.6"
          textAnchor="end"
        >
          budget limit
        </text>

        {/* Period reset markers */}
        {boundaryXs.map(({ x, date }) => (
          <g key={date}>
            <line
              x1={x}
              y1={0}
              x2={x}
              y2={cH}
              stroke={accent}
              strokeWidth="1"
              opacity="0.22"
              strokeDasharray="3 3"
            />
            <text x={x + 2} y={7} fontSize="5" fill={labelC} opacity="0.55">
              ↺ reset
            </text>
          </g>
        ))}

        {/* Historical segment areas + lines (faded) */}
        {actualSegments.slice(0, -1).map(
          (seg, si) =>
            seg.length > 1 && (
              <g key={si} opacity={0.4}>
                <AnimArea
                  as="path"
                  d={svgArea(seg)}
                  fill="url(#fg-fill-actual)"
                  $delay={`${0.05 * si}s`}
                />
                <AnimPath
                  d={svgLine(seg)}
                  fill="none"
                  stroke={actual}
                  strokeWidth="1.5"
                  strokeLinecap="round"
                  strokeLinejoin="round"
                  pathLength="1"
                  $dur={`${lineDur * 0.6}s`}
                  $delay={`${0.05 * si}s`}
                  $dasharray="1"
                />
              </g>
            )
        )}

        {/* Current period area */}
        {actualSegments.length > 0 && actualSegments[actualSegments.length - 1].length > 1 && (
          <AnimArea
            as="path"
            d={svgArea(actualSegments[actualSegments.length - 1])}
            fill="url(#fg-fill-actual)"
            $delay="0.15s"
          />
        )}

        {/* Projected area */}
        {projPts.length > 1 && (
          <AnimArea
            as="path"
            d={svgArea(projPts)}
            fill="url(#fg-fill-proj)"
            $delay={`${lineDur * 0.55}s`}
          />
        )}

        {/* Today divider */}
        {todayPt && todayPt.x > 0 && (
          <line
            x1={todayPt.x}
            y1={0}
            x2={todayPt.x}
            y2={cH}
            stroke={accent}
            strokeWidth="1"
            opacity="0.15"
            strokeDasharray="3 3"
          />
        )}

        {/* Current period actual line */}
        {actualSegments.length > 0 && actualSegments[actualSegments.length - 1].length > 1 && (
          <AnimPath
            d={svgLine(actualSegments[actualSegments.length - 1])}
            fill="none"
            stroke="url(#fg-stroke-grad)"
            strokeWidth="2.5"
            strokeLinecap="round"
            strokeLinejoin="round"
            pathLength="1"
            $dur={`${lineDur}s`}
            $delay="0.1s"
            $dasharray="1"
          />
        )}

        {/* Projected line */}
        {projPts.length > 1 && (
          <AnimProjPath
            d={svgLine(projPts)}
            fill="none"
            stroke={accent}
            strokeWidth="1.5"
            strokeLinecap="round"
            $dur="0.5s"
            $delay={`${lineDur * 0.52}s`}
          />
        )}

        {/* Staggered dots on current period */}
        {actualSegments.length > 0 &&
          actualSegments[actualSegments.length - 1].map((pt, i) => {
            const delay = `${0.1 + (i / Math.max(actualSegments[actualSegments.length - 1].length - 1, 1)) * lineDur}s`;
            return <AnimDot key={i} cx={pt.x} cy={pt.y} r={2.5} fill={accent} $delay={delay} />;
          })}

        {/* Today: glow ring + solid dot */}
        {todayPt && (
          <>
            <PulseDot
              cx={todayPt.x}
              cy={todayPt.y}
              r={6}
              fill={accent}
              opacity={0.2}
              filter="url(#fg-glow)"
              $delay={`${lineDur + 0.1}s`}
            />
            <TodayDot
              cx={todayPt.x}
              cy={todayPt.y}
              r={4.5}
              fill={accent}
              $delay={`${lineDur + 0.05}s`}
            />
            <TodayDot
              cx={todayPt.x}
              cy={todayPt.y}
              r={2}
              fill="white"
              $delay={`${lineDur + 0.05}s`}
            />
          </>
        )}

        {/* Crosshair on hover */}
        {hover && (
          <>
            <line
              x1={hover.x}
              y1={0}
              x2={hover.x}
              y2={cH}
              stroke={accent}
              strokeWidth="1"
              opacity="0.22"
              strokeDasharray="3 3"
            />
            <circle cx={hover.x} cy={hover.y} r={5} fill={accent} opacity="0.9" />
            <circle cx={hover.x} cy={hover.y} r={2.5} fill="white" opacity="0.9" />
          </>
        )}

        {/* Axes */}
        <line x1={0} y1={cH} x2={cW} y2={cH} stroke={gridC} strokeWidth="1" />
        {xLabels.map((l, i) => (
          <text key={i} x={l.x} y={cH + 18} textAnchor="middle" fontSize="5.5" fill={labelC}>
            {l.label}
          </text>
        ))}
        {yTicks.map((t, i) => (
          <text key={i} x={-6} y={t.y + 3} textAnchor="end" fontSize="5.5" fill={labelC}>
            {t.label}
          </text>
        ))}

        {/* Mouse overlay */}
        <rect
          x={0}
          y={0}
          width={cW}
          height={cH}
          fill="transparent"
          style={{ cursor: 'crosshair' }}
          onMouseMove={handleMouseMove}
          onMouseLeave={() => setHover(null)}
        />

        {/* Tooltip */}
        {hover && (
          <g transform={`translate(${ttX},${ttY})`} style={{ pointerEvents: 'none' }}>
            <rect
              rx="5"
              ry="5"
              width="96"
              height="38"
              fill="var(--color-card-bg)"
              stroke={colors.border}
              strokeWidth="1"
            />
            <text x="10" y="15" fontSize="5.5" fill={labelC}>
              {hover.date} · {hover.type}
            </text>
            <text x="10" y="29" fontSize="8" fontWeight="600" fill="var(--color-foreground)">
              {fmt(hover.value)}
            </text>
          </g>
        )}
      </g>
    </svg>
  );
}

// ── Export ────────────────────────────────────────────────────────────────────

interface ForecastSectionProps {
  forecast: BudgetForecast;
}

export function ForecastSection({ forecast }: ForecastSectionProps) {
  const confidenceLabel =
    forecast.confidence >= 0.7
      ? 'High confidence'
      : forecast.confidence >= 0.4
        ? 'Moderate confidence'
        : 'Low confidence';

  const noUsage = forecast.current_usage_usd === 0 && forecast.burn_rate_usd_per_day === 0;

  const daysLabel =
    forecast.days_until_exhausted === -1
      ? noUsage
        ? 'No usage yet'
        : 'On track'
      : forecast.days_until_exhausted < 1
        ? `~${Math.round(forecast.days_until_exhausted * 24)}h`
        : `${Math.ceil(forecast.days_until_exhausted)} days`;

  const overBudget = forecast.projected_spend_usd > forecast.budget_amount_usd;

  return (
    <Section>
      <SectionHeader>
        <SectionTitle>Spending Forecast</SectionTitle>
        <ConfidenceBadge $c={forecast.confidence}>
          {confidenceLabel} ({Math.round(forecast.confidence * 100)}%)
        </ConfidenceBadge>
      </SectionHeader>

      <StatsGrid>
        <StatCard>
          <StatLabel>
            <Tooltip text="Total projected spend by end of this budget period, based on current burn rate.">
              Projected Spend
            </Tooltip>
          </StatLabel>
          <StatValue style={{ color: overBudget ? 'var(--color-error)' : undefined }}>
            {fmt(forecast.projected_spend_usd)}
          </StatValue>
          <StatSub>of {fmt(forecast.budget_amount_usd)} budget</StatSub>
        </StatCard>
        <StatCard>
          <StatLabel>
            <Tooltip
              text={`Average daily spend based on recent completed days.\nStabilises once a full day of data is available.`}
            >
              Burn Rate
            </Tooltip>
          </StatLabel>
          <StatValue>{fmt(forecast.burn_rate_usd_per_day)}</StatValue>
          <StatSub>per day</StatSub>
        </StatCard>
        <StatCard>
          <StatLabel>
            <Tooltip
              text={`Days remaining before the budget limit is reached within this period.\nShows 'On track' if projected spend stays under the limit.`}
            >
              Budget Exhausted In
            </Tooltip>
          </StatLabel>
          <StatValue
            style={{
              color:
                forecast.days_until_exhausted === -1
                  ? noUsage
                    ? undefined
                    : 'var(--color-success)'
                  : forecast.days_until_exhausted < 3
                    ? 'var(--color-error)'
                    : forecast.days_until_exhausted < 7
                      ? 'var(--color-warning)'
                      : undefined,
            }}
          >
            {daysLabel}
          </StatValue>
          <StatSub>within this period</StatSub>
        </StatCard>
        <StatCard>
          <StatLabel>
            <Tooltip text="Actual spend recorded so far in the current budget period.">
              Current Usage
            </Tooltip>
          </StatLabel>
          <StatValue>{fmt(forecast.current_usage_usd)}</StatValue>
          <StatSub>
            {forecast.budget_amount_usd > 0
              ? `${((forecast.current_usage_usd / forecast.budget_amount_usd) * 100).toFixed(1)}% of budget`
              : '—'}
          </StatSub>
        </StatCard>
      </StatsGrid>

      <ChartCard>
        <ChartLegend>
          {(forecast.time_series || []).some(p => p.actual_usd != null) && (
            <LegendItem>
              <LegendDash color="var(--color-chart-actual)" />
              Actual spend
            </LegendItem>
          )}
          <LegendItem>
            <LegendDash color="var(--color-chart-accent)" dashed />
            Projected
          </LegendItem>
          <LegendItem>
            <LegendDash color="var(--color-chart-error)" dashed $opacity={0.5} />
            Budget limit
          </LegendItem>
          {(forecast.period_boundaries || []).length > 0 && (
            <LegendItem>
              <LegendDash color="var(--color-chart-accent)" dashed $opacity={0.3} />
              Period reset
            </LegendItem>
          )}
        </ChartLegend>
        <ForecastChart forecast={forecast} />
      </ChartCard>
    </Section>
  );
}
