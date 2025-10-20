/**
 * Strength metrics section showing entropy distribution and crack time estimates.
 */
import React from 'react';
import {
  Paper,
  Typography,
  Table,
  TableBody,
  TableCell,
  TableContainer,
  TableHead,
  TableRow,
  Box,
} from '@mui/material';
import { StrengthStats } from '../../types/analytics';

interface StrengthMetricsSectionProps {
  data: StrengthStats;
}

export default function StrengthMetricsSection({ data }: StrengthMetricsSectionProps) {
  const formatSpeed = (hps: number): string => {
    if (hps >= 1000000000) return `${(hps / 1000000000).toFixed(2)} GH/s`;
    if (hps >= 1000000) return `${(hps / 1000000).toFixed(2)} MH/s`;
    if (hps >= 1000) return `${(hps / 1000).toFixed(2)} KH/s`;
    return `${hps.toFixed(0)} H/s`;
  };

  return (
    <Paper sx={{ p: 3, mb: 3 }}>
      <Typography variant="h5" gutterBottom>
        Strength Metrics
      </Typography>

      {/* Entropy Distribution */}
      <Box sx={{ mb: 4 }}>
        <Typography variant="h6" gutterBottom>
          Entropy Distribution
        </Typography>
        <Typography variant="body2" color="text.secondary" sx={{ mb: 2 }}>
          Password strength categorization based on Shannon entropy
        </Typography>
        <TableContainer>
          <Table size="small">
            <TableHead>
              <TableRow>
                <TableCell>Entropy Level</TableCell>
                <TableCell>Range</TableCell>
                <TableCell align="right">Count</TableCell>
                <TableCell align="right">Percentage</TableCell>
              </TableRow>
            </TableHead>
            <TableBody>
              <TableRow>
                <TableCell>Low</TableCell>
                <TableCell>&lt;78 bits</TableCell>
                <TableCell align="right">{data.entropy_distribution.low.count.toLocaleString()}</TableCell>
                <TableCell align="right">{data.entropy_distribution.low.percentage.toFixed(2)}%</TableCell>
              </TableRow>
              <TableRow>
                <TableCell>Moderate</TableCell>
                <TableCell>78-127 bits</TableCell>
                <TableCell align="right">{data.entropy_distribution.moderate.count.toLocaleString()}</TableCell>
                <TableCell align="right">{data.entropy_distribution.moderate.percentage.toFixed(2)}%</TableCell>
              </TableRow>
              <TableRow>
                <TableCell>High</TableCell>
                <TableCell>128+ bits</TableCell>
                <TableCell align="right">{data.entropy_distribution.high.count.toLocaleString()}</TableCell>
                <TableCell align="right">{data.entropy_distribution.high.percentage.toFixed(2)}%</TableCell>
              </TableRow>
            </TableBody>
          </Table>
        </TableContainer>
      </Box>

      {/* Crack Time Estimates */}
      <Box>
        <Typography variant="h6" gutterBottom>
          Crack Time Estimates by Speed
        </Typography>
        <Typography variant="body2" color="text.secondary" sx={{ mb: 2 }}>
          Percentage of passwords crackable within time periods at different speed levels
        </Typography>
        <TableContainer>
          <Table size="small">
            <TableHead>
              <TableRow>
                <TableCell>Speed Level</TableCell>
                <TableCell align="right">Speed (H/s)</TableCell>
                <TableCell align="right">&lt;1 Hour</TableCell>
                <TableCell align="right">&lt;1 Day</TableCell>
                <TableCell align="right">&lt;1 Week</TableCell>
                <TableCell align="right">&lt;1 Month</TableCell>
                <TableCell align="right">&lt;6 Months</TableCell>
                <TableCell align="right">&lt;1 Year</TableCell>
                <TableCell align="right">&gt;1 Year</TableCell>
              </TableRow>
            </TableHead>
            <TableBody>
              <TableRow>
                <TableCell>50% Speed</TableCell>
                <TableCell align="right">{formatSpeed(data.crack_time_estimates.speed_50_percent.speed_hps)}</TableCell>
                <TableCell align="right">{data.crack_time_estimates.speed_50_percent.percent_under_1_hour.toFixed(2)}%</TableCell>
                <TableCell align="right">{data.crack_time_estimates.speed_50_percent.percent_under_1_day.toFixed(2)}%</TableCell>
                <TableCell align="right">{data.crack_time_estimates.speed_50_percent.percent_under_1_week.toFixed(2)}%</TableCell>
                <TableCell align="right">{data.crack_time_estimates.speed_50_percent.percent_under_1_month.toFixed(2)}%</TableCell>
                <TableCell align="right">{data.crack_time_estimates.speed_50_percent.percent_under_6_months.toFixed(2)}%</TableCell>
                <TableCell align="right">{data.crack_time_estimates.speed_50_percent.percent_under_1_year.toFixed(2)}%</TableCell>
                <TableCell align="right">{data.crack_time_estimates.speed_50_percent.percent_over_1_year.toFixed(2)}%</TableCell>
              </TableRow>
              <TableRow>
                <TableCell>75% Speed</TableCell>
                <TableCell align="right">{formatSpeed(data.crack_time_estimates.speed_75_percent.speed_hps)}</TableCell>
                <TableCell align="right">{data.crack_time_estimates.speed_75_percent.percent_under_1_hour.toFixed(2)}%</TableCell>
                <TableCell align="right">{data.crack_time_estimates.speed_75_percent.percent_under_1_day.toFixed(2)}%</TableCell>
                <TableCell align="right">{data.crack_time_estimates.speed_75_percent.percent_under_1_week.toFixed(2)}%</TableCell>
                <TableCell align="right">{data.crack_time_estimates.speed_75_percent.percent_under_1_month.toFixed(2)}%</TableCell>
                <TableCell align="right">{data.crack_time_estimates.speed_75_percent.percent_under_6_months.toFixed(2)}%</TableCell>
                <TableCell align="right">{data.crack_time_estimates.speed_75_percent.percent_under_1_year.toFixed(2)}%</TableCell>
                <TableCell align="right">{data.crack_time_estimates.speed_75_percent.percent_over_1_year.toFixed(2)}%</TableCell>
              </TableRow>
              <TableRow>
                <TableCell>100% Speed (Average)</TableCell>
                <TableCell align="right">{formatSpeed(data.crack_time_estimates.speed_100_percent.speed_hps)}</TableCell>
                <TableCell align="right">{data.crack_time_estimates.speed_100_percent.percent_under_1_hour.toFixed(2)}%</TableCell>
                <TableCell align="right">{data.crack_time_estimates.speed_100_percent.percent_under_1_day.toFixed(2)}%</TableCell>
                <TableCell align="right">{data.crack_time_estimates.speed_100_percent.percent_under_1_week.toFixed(2)}%</TableCell>
                <TableCell align="right">{data.crack_time_estimates.speed_100_percent.percent_under_1_month.toFixed(2)}%</TableCell>
                <TableCell align="right">{data.crack_time_estimates.speed_100_percent.percent_under_6_months.toFixed(2)}%</TableCell>
                <TableCell align="right">{data.crack_time_estimates.speed_100_percent.percent_under_1_year.toFixed(2)}%</TableCell>
                <TableCell align="right">{data.crack_time_estimates.speed_100_percent.percent_over_1_year.toFixed(2)}%</TableCell>
              </TableRow>
              <TableRow>
                <TableCell>150% Speed</TableCell>
                <TableCell align="right">{formatSpeed(data.crack_time_estimates.speed_150_percent.speed_hps)}</TableCell>
                <TableCell align="right">{data.crack_time_estimates.speed_150_percent.percent_under_1_hour.toFixed(2)}%</TableCell>
                <TableCell align="right">{data.crack_time_estimates.speed_150_percent.percent_under_1_day.toFixed(2)}%</TableCell>
                <TableCell align="right">{data.crack_time_estimates.speed_150_percent.percent_under_1_week.toFixed(2)}%</TableCell>
                <TableCell align="right">{data.crack_time_estimates.speed_150_percent.percent_under_1_month.toFixed(2)}%</TableCell>
                <TableCell align="right">{data.crack_time_estimates.speed_150_percent.percent_under_6_months.toFixed(2)}%</TableCell>
                <TableCell align="right">{data.crack_time_estimates.speed_150_percent.percent_under_1_year.toFixed(2)}%</TableCell>
                <TableCell align="right">{data.crack_time_estimates.speed_150_percent.percent_over_1_year.toFixed(2)}%</TableCell>
              </TableRow>
              <TableRow>
                <TableCell>200% Speed</TableCell>
                <TableCell align="right">{formatSpeed(data.crack_time_estimates.speed_200_percent.speed_hps)}</TableCell>
                <TableCell align="right">{data.crack_time_estimates.speed_200_percent.percent_under_1_hour.toFixed(2)}%</TableCell>
                <TableCell align="right">{data.crack_time_estimates.speed_200_percent.percent_under_1_day.toFixed(2)}%</TableCell>
                <TableCell align="right">{data.crack_time_estimates.speed_200_percent.percent_under_1_week.toFixed(2)}%</TableCell>
                <TableCell align="right">{data.crack_time_estimates.speed_200_percent.percent_under_1_month.toFixed(2)}%</TableCell>
                <TableCell align="right">{data.crack_time_estimates.speed_200_percent.percent_under_6_months.toFixed(2)}%</TableCell>
                <TableCell align="right">{data.crack_time_estimates.speed_200_percent.percent_under_1_year.toFixed(2)}%</TableCell>
                <TableCell align="right">{data.crack_time_estimates.speed_200_percent.percent_over_1_year.toFixed(2)}%</TableCell>
              </TableRow>
            </TableBody>
          </Table>
        </TableContainer>
      </Box>
    </Paper>
  );
}
