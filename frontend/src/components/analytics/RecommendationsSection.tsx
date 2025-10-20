/**
 * Recommendations section showing auto-generated password policy recommendations.
 */
import React from 'react';
import {
  Paper,
  Typography,
  List,
  ListItem,
  ListItemText,
  Alert,
  Chip,
  Box,
} from '@mui/material';
import { Recommendation } from '../../types/analytics';

interface RecommendationsSectionProps {
  data: Recommendation[];
}

export default function RecommendationsSection({ data }: RecommendationsSectionProps) {
  if (data.length === 0) {
    return (
      <Paper sx={{ p: 3, mb: 3 }}>
        <Typography variant="h5" gutterBottom>
          Recommendations
        </Typography>
        <Alert severity="success">
          No specific recommendations at this time. Password policies appear adequate.
        </Alert>
      </Paper>
    );
  }

  const getSeverityColor = (severity: string) => {
    switch (severity) {
      case 'CRITICAL': return 'error';
      case 'HIGH': return 'warning';
      case 'MEDIUM': return 'info';
      case 'INFO': return 'default';
      default: return 'default';
    }
  };

  return (
    <Paper sx={{ p: 3, mb: 3 }}>
      <Typography variant="h5" gutterBottom>
        Recommendations
      </Typography>
      <Typography variant="body2" color="text.secondary" sx={{ mb: 2 }}>
        Suggested password policy improvements based on analysis
      </Typography>

      <List>
        {data.map((recommendation, index) => (
          <ListItem key={index}>
            <Box sx={{ display: 'flex', alignItems: 'center', width: '100%', gap: 2 }}>
              <Chip
                label={recommendation.severity}
                color={getSeverityColor(recommendation.severity) as any}
                size="small"
              />
              <ListItemText
                primary={recommendation.message}
                secondary={`${recommendation.count.toLocaleString()} passwords (${recommendation.percentage.toFixed(2)}%)`}
                primaryTypographyProps={{
                  variant: 'body1',
                  sx: { fontWeight: 500 },
                }}
              />
            </Box>
          </ListItem>
        ))}
      </List>
    </Paper>
  );
}
