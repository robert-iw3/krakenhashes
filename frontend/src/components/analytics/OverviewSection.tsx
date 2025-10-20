/**
 * Overview section showing high-level statistics and hash mode breakdown.
 */
import React from 'react';
import {
  Paper,
  Typography,
  Grid,
  Card,
  CardContent,
  Table,
  TableBody,
  TableCell,
  TableContainer,
  TableHead,
  TableRow,
  Box,
} from '@mui/material';
import { AnalyticsReport, AnalyticsData } from '../../types/analytics';

interface OverviewSectionProps {
  report: AnalyticsReport;
  data: AnalyticsData;
}

export default function OverviewSection({ report, data }: OverviewSectionProps) {
  const crackPercentage = data.overview.crack_percentage.toFixed(2);

  return (
    <Paper sx={{ p: 3, mb: 3 }}>
      <Typography variant="h5" gutterBottom>
        Overview
      </Typography>

      {/* Summary Cards */}
      <Grid container spacing={3} sx={{ mb: 3 }}>
        <Grid item xs={12} md={3}>
          <Card>
            <CardContent>
              <Typography color="text.secondary" gutterBottom>
                Total Hashes
              </Typography>
              <Typography variant="h4">
                {data.overview.total_hashes.toLocaleString()}
              </Typography>
            </CardContent>
          </Card>
        </Grid>
        <Grid item xs={12} md={3}>
          <Card>
            <CardContent>
              <Typography color="text.secondary" gutterBottom>
                Cracked
              </Typography>
              <Typography variant="h4" color="success.main">
                {data.overview.total_cracked.toLocaleString()}
              </Typography>
            </CardContent>
          </Card>
        </Grid>
        <Grid item xs={12} md={3}>
          <Card>
            <CardContent>
              <Typography color="text.secondary" gutterBottom>
                Crack Rate
              </Typography>
              <Typography variant="h4" color="primary">
                {crackPercentage}%
              </Typography>
            </CardContent>
          </Card>
        </Grid>
        <Grid item xs={12} md={3}>
          <Card>
            <CardContent>
              <Typography color="text.secondary" gutterBottom>
                Hashlists Analyzed
              </Typography>
              <Typography variant="h4">
                {report.total_hashlists}
              </Typography>
            </CardContent>
          </Card>
        </Grid>
      </Grid>

      {/* Hash Mode Breakdown */}
      <Box>
        <Typography variant="h6" gutterBottom>
          Hash Mode Breakdown
        </Typography>
        <TableContainer>
          <Table>
            <TableHead>
              <TableRow>
                <TableCell>Hash Type</TableCell>
                <TableCell align="right">Total Hashes</TableCell>
                <TableCell align="right">Cracked</TableCell>
                <TableCell align="right">Percentage</TableCell>
              </TableRow>
            </TableHead>
            <TableBody>
              {data.overview.hash_modes.map((stat) => (
                <TableRow key={stat.mode_id}>
                  <TableCell>{stat.mode_name}</TableCell>
                  <TableCell align="right">{stat.total.toLocaleString()}</TableCell>
                  <TableCell align="right">{stat.cracked.toLocaleString()}</TableCell>
                  <TableCell align="right">{stat.percentage.toFixed(2)}%</TableCell>
                </TableRow>
              ))}
              {/* Totals Row */}
              <TableRow sx={{ fontWeight: 'bold', backgroundColor: 'action.hover' }}>
                <TableCell>
                  <strong>Total</strong>
                </TableCell>
                <TableCell align="right">
                  <strong>{data.overview.total_hashes.toLocaleString()}</strong>
                </TableCell>
                <TableCell align="right">
                  <strong>{data.overview.total_cracked.toLocaleString()}</strong>
                </TableCell>
                <TableCell align="right">
                  <strong>{crackPercentage}%</strong>
                </TableCell>
              </TableRow>
            </TableBody>
          </Table>
        </TableContainer>
      </Box>
    </Paper>
  );
}
