/**
 * Custom patterns section showing organization name pattern matches.
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
} from '@mui/material';
import { CustomPatternStats } from '../../types/analytics';
import { threeColumnTableStyles } from './tableStyles';

interface CustomPatternsSectionProps {
  data: CustomPatternStats;
}

export default function CustomPatternsSection({ data }: CustomPatternsSectionProps) {
  const patterns = Object.entries(data.patterns_detected).filter(([_, value]) => value.count > 0);

  if (patterns.length === 0) {
    return null;
  }

  return (
    <Paper sx={{ p: 3, mb: 3 }}>
      <Typography variant="h5" gutterBottom>
        Organization Name Patterns
      </Typography>
      <Typography variant="body2" color="text.secondary" sx={{ mb: 2 }}>
        Passwords containing organization name variations
      </Typography>

      <TableContainer>
        <Table size="small">
          <TableHead>
            <TableRow>
              <TableCell sx={threeColumnTableStyles.labelCell}>Pattern</TableCell>
              <TableCell sx={threeColumnTableStyles.countCell}>Count</TableCell>
              <TableCell sx={threeColumnTableStyles.percentageCell}>Percentage</TableCell>
            </TableRow>
          </TableHead>
          <TableBody>
            {patterns.map(([patternName, stats], index) => (
              <TableRow key={index}>
                <TableCell sx={threeColumnTableStyles.labelCell}>{patternName}</TableCell>
                <TableCell sx={threeColumnTableStyles.countCell}>{stats.count.toLocaleString()}</TableCell>
                <TableCell sx={threeColumnTableStyles.percentageCell}>{stats.percentage.toFixed(2)}%</TableCell>
              </TableRow>
            ))}
          </TableBody>
        </Table>
      </TableContainer>
    </Paper>
  );
}
