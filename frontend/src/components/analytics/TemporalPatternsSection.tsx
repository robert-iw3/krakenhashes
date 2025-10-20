/**
 * Temporal patterns section showing years, months, and seasons in passwords.
 */
import React, { useMemo } from 'react';
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
import { TemporalStats } from '../../types/analytics';
import { threeColumnTableStyles } from './tableStyles';

interface TemporalPatternsSectionProps {
  data: TemporalStats;
}

export default function TemporalPatternsSection({ data }: TemporalPatternsSectionProps) {
  const years = useMemo(() =>
    Object.entries(data.year_breakdown).filter(([_, value]) => value.count > 0),
    [data.year_breakdown]
  );

  const hasData = years.length > 0 || data.contains_year.count > 0 ||
                  data.contains_month.count > 0 || data.contains_season.count > 0;

  if (!hasData) {
    return null;
  }

  return (
    <Paper sx={{ p: 3, mb: 3 }}>
      <Typography variant="h5" gutterBottom>
        Temporal Patterns
      </Typography>
      <Typography variant="body2" color="text.secondary" sx={{ mb: 2 }}>
        Date-related patterns found in passwords
      </Typography>

      {/* Summary */}
      <TableContainer sx={{ mb: 3 }}>
        <Table size="small">
          <TableHead>
            <TableRow>
              <TableCell sx={threeColumnTableStyles.labelCell}>Pattern Type</TableCell>
              <TableCell sx={threeColumnTableStyles.countCell}>Count</TableCell>
              <TableCell sx={threeColumnTableStyles.percentageCell}>Percentage</TableCell>
            </TableRow>
          </TableHead>
          <TableBody>
            {data.contains_year.count > 0 && (
              <TableRow>
                <TableCell sx={threeColumnTableStyles.labelCell}>Contains Year</TableCell>
                <TableCell sx={threeColumnTableStyles.countCell}>{data.contains_year.count.toLocaleString()}</TableCell>
                <TableCell sx={threeColumnTableStyles.percentageCell}>{data.contains_year.percentage.toFixed(2)}%</TableCell>
              </TableRow>
            )}
            {data.contains_month.count > 0 && (
              <TableRow>
                <TableCell sx={threeColumnTableStyles.labelCell}>Contains Month</TableCell>
                <TableCell sx={threeColumnTableStyles.countCell}>{data.contains_month.count.toLocaleString()}</TableCell>
                <TableCell sx={threeColumnTableStyles.percentageCell}>{data.contains_month.percentage.toFixed(2)}%</TableCell>
              </TableRow>
            )}
            {data.contains_season.count > 0 && (
              <TableRow>
                <TableCell sx={threeColumnTableStyles.labelCell}>Contains Season</TableCell>
                <TableCell sx={threeColumnTableStyles.countCell}>{data.contains_season.count.toLocaleString()}</TableCell>
                <TableCell sx={threeColumnTableStyles.percentageCell}>{data.contains_season.percentage.toFixed(2)}%</TableCell>
              </TableRow>
            )}
          </TableBody>
        </Table>
      </TableContainer>

      {/* Year Breakdown */}
      {years.length > 0 && (
        <Box>
          <Typography variant="h6" gutterBottom>
            Year Breakdown
          </Typography>
          <TableContainer>
            <Table size="small">
              <TableHead>
                <TableRow>
                  <TableCell sx={threeColumnTableStyles.labelCell}>Year</TableCell>
                  <TableCell sx={threeColumnTableStyles.countCell}>Count</TableCell>
                  <TableCell sx={threeColumnTableStyles.percentageCell}>Percentage</TableCell>
                </TableRow>
              </TableHead>
              <TableBody>
                {years.map(([year, stats]) => (
                  <TableRow key={year}>
                    <TableCell sx={threeColumnTableStyles.labelCell}>{year}</TableCell>
                    <TableCell sx={threeColumnTableStyles.countCell}>{stats.count.toLocaleString()}</TableCell>
                    <TableCell sx={threeColumnTableStyles.percentageCell}>{stats.percentage.toFixed(2)}%</TableCell>
                  </TableRow>
                ))}
              </TableBody>
            </Table>
          </TableContainer>
        </Box>
      )}
    </Paper>
  );
}
