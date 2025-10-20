/**
 * Positional analysis section showing uppercase start and numbers/special at end.
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
import { PositionalStats } from '../../types/analytics';
import { threeColumnTableStyles } from './tableStyles';

interface PositionalStatsSectionProps {
  data: PositionalStats;
}

export default function PositionalStatsSection({ data }: PositionalStatsSectionProps) {
  const hasData = data.starts_uppercase.count > 0 || data.ends_number.count > 0 || data.ends_special.count > 0;

  if (!hasData) {
    return null;
  }

  return (
    <Paper sx={{ p: 3, mb: 3 }}>
      <Typography variant="h5" gutterBottom>
        Positional Analysis
      </Typography>
      <Typography variant="body2" color="text.secondary" sx={{ mb: 2 }}>
        Common positional patterns in passwords
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
            {data.starts_uppercase.count > 0 && (
              <TableRow>
                <TableCell sx={threeColumnTableStyles.labelCell}>Starts with Uppercase</TableCell>
                <TableCell sx={threeColumnTableStyles.countCell}>{data.starts_uppercase.count.toLocaleString()}</TableCell>
                <TableCell sx={threeColumnTableStyles.percentageCell}>{data.starts_uppercase.percentage.toFixed(2)}%</TableCell>
              </TableRow>
            )}
            {data.ends_number.count > 0 && (
              <TableRow>
                <TableCell sx={threeColumnTableStyles.labelCell}>Ends with Number</TableCell>
                <TableCell sx={threeColumnTableStyles.countCell}>{data.ends_number.count.toLocaleString()}</TableCell>
                <TableCell sx={threeColumnTableStyles.percentageCell}>{data.ends_number.percentage.toFixed(2)}%</TableCell>
              </TableRow>
            )}
            {data.ends_special.count > 0 && (
              <TableRow>
                <TableCell sx={threeColumnTableStyles.labelCell}>Ends with Special Character</TableCell>
                <TableCell sx={threeColumnTableStyles.countCell}>{data.ends_special.count.toLocaleString()}</TableCell>
                <TableCell sx={threeColumnTableStyles.percentageCell}>{data.ends_special.percentage.toFixed(2)}%</TableCell>
              </TableRow>
            )}
          </TableBody>
        </Table>
      </TableContainer>
    </Paper>
  );
}
