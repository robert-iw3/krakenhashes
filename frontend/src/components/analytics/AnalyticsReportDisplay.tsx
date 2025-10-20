/**
 * Main display component for analytics reports.
 * Handles status-aware rendering and coordinates all section components.
 */
import React from 'react';
import {
  Box,
  Paper,
  Typography,
  Alert,
  AlertTitle,
  Button,
  LinearProgress,
} from '@mui/material';
import { Replay as RetryIcon, Delete as DeleteIcon } from '@mui/icons-material';
import { AnalyticsReport } from '../../types/analytics';
import OverviewSection from './OverviewSection';
import LengthDistributionSection from './LengthDistributionSection';
import ComplexityAnalysisSection from './ComplexityAnalysisSection';
import PositionalAnalysisSection from './PositionalAnalysisSection';
import PatternDetectionSection from './PatternDetectionSection';
import UsernameCorrelationSection from './UsernameCorrelationSection';
import PasswordReuseSection from './PasswordReuseSection';
import TemporalPatternsSection from './TemporalPatternsSection';
import MaskAnalysisSection from './MaskAnalysisSection';
import CustomPatternsSection from './CustomPatternsSection';
import StrengthMetricsSection from './StrengthMetricsSection';
import TopPasswordsSection from './TopPasswordsSection';
import RecommendationsSection from './RecommendationsSection';

interface AnalyticsReportDisplayProps {
  report: AnalyticsReport;
  status: string;
  onRetry?: () => void;
  onDelete?: () => void;
}

export default function AnalyticsReportDisplay({
  report,
  status,
  onRetry,
  onDelete,
}: AnalyticsReportDisplayProps) {
  // Render status-specific UI
  const renderStatusUI = () => {
    switch (status) {
      case 'queued':
        return (
          <Alert severity="info" sx={{ mb: 3 }}>
            <AlertTitle>Pending Report Generation</AlertTitle>
            Your report is queued for processing. Position: {report.queue_position || 'N/A'}
            <LinearProgress sx={{ mt: 2 }} />
          </Alert>
        );

      case 'processing':
        return (
          <Alert severity="info" sx={{ mb: 3 }}>
            <AlertTitle>Report is Still Generating</AlertTitle>
            Your report is currently being processed. This may take several minutes depending on the dataset size.
            <LinearProgress sx={{ mt: 2 }} />
          </Alert>
        );

      case 'failed':
        return (
          <Alert
            severity="error"
            sx={{ mb: 3 }}
            action={
              <Box>
                {onRetry && (
                  <Button color="inherit" size="small" startIcon={<RetryIcon />} onClick={onRetry}>
                    Retry
                  </Button>
                )}
                {onDelete && (
                  <Button color="inherit" size="small" startIcon={<DeleteIcon />} onClick={onDelete}>
                    Delete
                  </Button>
                )}
              </Box>
            }
          >
            <AlertTitle>Report Generation Failed</AlertTitle>
            {report.error_message || 'An error occurred while generating the report.'}
          </Alert>
        );

      default:
        return null;
    }
  };

  // Don't render full report if not completed
  if (status !== 'completed' || !report.analytics_data) {
    return (
      <Paper sx={{ p: 3 }}>
        {renderStatusUI()}
        <Typography variant="body2" color="text.secondary">
          Report ID: {report.id}
        </Typography>
      </Paper>
    );
  }

  const data = report.analytics_data;

  return (
    <Box>
      {/* Status indicator */}
      {renderStatusUI()}

      {/* Report ID for debugging */}
      <Typography variant="body2" color="text.secondary" sx={{ mb: 2 }}>
        Report ID: {report.id}
      </Typography>

      {/* Overview Section - Full Width */}
      <OverviewSection report={report} data={data} />

      {/* Masonry-style layout using CSS columns - cards flow vertically to fill space */}
      <Box
        sx={{
          columnCount: { xs: 1, md: 2 },
          columnGap: 3,
          '& > *': {
            breakInside: 'avoid',
            marginBottom: 3,
            display: 'inline-block',
            width: '100%',
          },
        }}
      >
        <LengthDistributionSection data={data.length_distribution} />

        <ComplexityAnalysisSection data={data.complexity_analysis} />

        <PositionalAnalysisSection data={data.positional_analysis} />

        <PatternDetectionSection data={data.pattern_detection} />

        <UsernameCorrelationSection data={data.username_correlation} />

        <TemporalPatternsSection data={data.temporal_patterns} />

        <MaskAnalysisSection data={data.mask_analysis} />

        {data.custom_patterns && Object.keys(data.custom_patterns.patterns_detected).length > 0 && (
          <CustomPatternsSection data={data.custom_patterns} />
        )}
      </Box>

      {/* Full-Width Sections Below Grid */}
      <StrengthMetricsSection data={data.strength_metrics} />

      <PasswordReuseSection data={data.password_reuse} />

      <TopPasswordsSection data={data.top_passwords} />

      <RecommendationsSection data={data.recommendations} />
    </Box>
  );
}
