import styled from '@emotion/styled';
import { colors, spacing, fontSize } from '../../styles';

const FooterContainer = styled.footer`
  padding: ${spacing[4]} ${spacing[8]};
  text-align: center;
  font-size: ${fontSize.xs};
  color: ${colors.dimForeground};
`;

function formatBuildTime(iso: string): string {
  return new Date(iso).toLocaleString('en-US', {
    month: 'short',
    day: 'numeric',
    year: 'numeric',
    hour: '2-digit',
    minute: '2-digit',
    hour12: false,
    timeZone: 'UTC',
    timeZoneName: 'short',
  });
}

export function Footer() {
  const version = __APP_VERSION__;
  const buildTime = formatBuildTime(__BUILD_TIME__);
  return (
    <FooterContainer>
      v{version} · Built {buildTime}
    </FooterContainer>
  );
}
