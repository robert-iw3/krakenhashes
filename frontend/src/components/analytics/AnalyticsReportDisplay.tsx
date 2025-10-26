/**
 * Main display component for analytics reports.
 * Handles status-aware rendering and coordinates all section components.
 */
import React, { useState } from 'react';
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
  const [selectedDomain, setSelectedDomain] = useState<string | null>(null);
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

  // Get the analytics data based on selected domain
  const getAnalyticsData = () => {
    if (!selectedDomain || !data.domain_analytics) {
      return data; // Return "All" data
    }

    // Find domain-specific analytics
    const domainData = data.domain_analytics.find((d) => d.domain === selectedDomain);
    if (!domainData) {
      return data; // Fallback to "All"
    }

    // Return domain-filtered analytics
    return {
      overview: domainData.overview,
      length_distribution: domainData.length_distribution,
      complexity_analysis: domainData.complexity_analysis,
      positional_analysis: domainData.positional_analysis,
      pattern_detection: domainData.pattern_detection,
      username_correlation: domainData.username_correlation,
      password_reuse: domainData.password_reuse,
      temporal_patterns: domainData.temporal_patterns,
      mask_analysis: domainData.mask_analysis,
      custom_patterns: domainData.custom_patterns,
      strength_metrics: domainData.strength_metrics,
      top_passwords: domainData.top_passwords,
      recommendations: data.recommendations, // Keep global recommendations
      domain_analytics: data.domain_analytics, // Keep for reference
    };
  };

  const filteredData = getAnalyticsData();

  return (
    <Box>
      {/* Status indicator */}
      {renderStatusUI()}

      {/* Report ID for debugging */}
      <Typography variant="body2" color="text.secondary" sx={{ mb: 2 }}>
        Report ID: {report.id}
      </Typography>

      {/* Overview Section - Full Width */}
      <OverviewSection
        report={report}
        data={data}
        filteredData={filteredData}
        selectedDomain={selectedDomain}
        onDomainChange={setSelectedDomain}
      />

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
        <LengthDistributionSection data={filteredData.length_distribution} />

        <ComplexityAnalysisSection data={filteredData.complexity_analysis} />

        <PositionalAnalysisSection data={filteredData.positional_analysis} />

        <PatternDetectionSection data={filteredData.pattern_detection} />

        <UsernameCorrelationSection data={filteredData.username_correlation} />

        <TemporalPatternsSection data={filteredData.temporal_patterns} />

        <MaskAnalysisSection data={filteredData.mask_analysis} />

        {filteredData.custom_patterns && Object.keys(filteredData.custom_patterns.patterns_detected).length > 0 && (
          <CustomPatternsSection data={filteredData.custom_patterns} />
        )}
      </Box>

      {/* Full-Width Sections Below Grid */}
      <StrengthMetricsSection data={filteredData.strength_metrics} />

      <PasswordReuseSection data={filteredData.password_reuse} />

      <TopPasswordsSection data={filteredData.top_passwords} />

      <RecommendationsSection data={filteredData.recommendations} />
    </Box>
  );
}
