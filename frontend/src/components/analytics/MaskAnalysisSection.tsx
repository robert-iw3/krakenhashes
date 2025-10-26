/**
 * Mask analysis section showing hashcat-style mask patterns.
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
} from '@mui/material';
import { MaskStats } from '../../types/analytics';
import { threeColumnTableStyles } from './tableStyles';

interface MaskAnalysisSectionProps {
  data: MaskStats;
}

export default function MaskAnalysisSection({ data }: MaskAnalysisSectionProps) {
  // Filter and sort masks by count
  const topMasks = useMemo(() => {
    return data.top_masks
      .filter(mask => mask.count > 0)
      .sort((a, b) => b.count - a.count)
      .slice(0, 20); // Show top 20 masks
  }, [data.top_masks]);

  if (topMasks.length === 0) {
    return null;
  }

  return (
    <Paper sx={{ p: 3, mb: 3 }}>
      <Typography variant="h5" gutterBottom>
        Mask Analysis
      </Typography>
      <Typography variant="body2" color="text.secondary" sx={{ mb: 2 }}>
        Top password patterns in hashcat mask format (?u=upper, ?l=lower, ?d=digit, ?s=special)
      </Typography>

      <TableContainer>
        <Table size="small">
          <TableHead>
            <TableRow>
              <TableCell sx={threeColumnTableStyles.labelCell}>Mask Pattern</TableCell>
              <TableCell sx={threeColumnTableStyles.countCell}>Count</TableCell>
              <TableCell sx={threeColumnTableStyles.percentageCell}>Percentage</TableCell>
            </TableRow>
          </TableHead>
          <TableBody>
            {topMasks.map((mask, index) => (
              <TableRow key={index}>
                <TableCell sx={{ ...threeColumnTableStyles.labelCell, fontFamily: 'monospace' }}>{mask.mask}</TableCell>
                <TableCell sx={threeColumnTableStyles.countCell}>{mask.count.toLocaleString()}</TableCell>
                <TableCell sx={threeColumnTableStyles.percentageCell}>{mask.percentage.toFixed(2)}%</TableCell>
              </TableRow>
            ))}
          </TableBody>
        </Table>
      </TableContainer>
    </Paper>
  );
}
