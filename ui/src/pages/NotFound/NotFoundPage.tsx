import { Link } from 'react-router-dom';
import styled from '@emotion/styled';
import { spacing, colors, fontSize } from '../../styles';
import { Button } from '../../components/common/Button';

const Container = styled.div`
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  min-height: 60vh;
  text-align: center;
  padding: ${spacing[8]};
`;

const ErrorCode = styled.h1`
  font-size: 120px;
  font-weight: 700;
  color: ${colors.mutedForeground};
  margin: 0;
  line-height: 1;
  opacity: 0.3;
`;

const Title = styled.h2`
  font-size: ${fontSize['2xl']};
  font-weight: 600;
  color: ${colors.foreground};
  margin: ${spacing[4]} 0 ${spacing[2]};
`;

const Description = styled.p`
  font-size: ${fontSize.md};
  color: ${colors.mutedForeground};
  margin: 0 0 ${spacing[6]};
  max-width: 400px;
`;

const Actions = styled.div`
  display: flex;
  gap: ${spacing[3]};
`;

export function NotFoundPage() {
  return (
    <Container>
      <ErrorCode>404</ErrorCode>
      <Title>Page not found</Title>
      <Description>The page you're looking for doesn't exist or has been moved.</Description>
      <Actions>
        <Link to="/">
          <Button>Go to Dashboard</Button>
        </Link>
        <Button variant="secondary" onClick={() => window.history.back()}>
          Go Back
        </Button>
      </Actions>
    </Container>
  );
}
