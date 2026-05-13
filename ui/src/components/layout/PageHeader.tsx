import styled from '@emotion/styled';
import { colors, spacing, fontSize } from '../../styles';

const Header = styled.header`
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: ${spacing[6]};
`;

const TitleSection = styled.div``;

const Title = styled.h1`
  font-size: ${fontSize['2xl']};
  font-weight: 600;
  color: ${colors.foreground};
`;

const Description = styled.p`
  font-size: ${fontSize.sm};
  color: ${colors.mutedForeground};
  margin-top: ${spacing[1]};
`;

const Actions = styled.div`
  display: flex;
  align-items: center;
  gap: ${spacing[3]};
`;

interface PageHeaderProps {
  title?: string;
  description?: string;
  children?: React.ReactNode;
}

export function PageHeader({ title, description, children }: PageHeaderProps) {
  return (
    <Header>
      <TitleSection>
        {title && <Title>{title}</Title>}
        {description && <Description>{description}</Description>}
      </TitleSection>
      {children && <Actions>{children}</Actions>}
    </Header>
  );
}
