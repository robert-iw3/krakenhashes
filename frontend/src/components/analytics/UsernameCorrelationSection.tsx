/**
 * Username correlation section showing password-username relationships.
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
import { UsernameStats } from '../../types/analytics';
import { threeColumnTableStyles } from './tableStyles';

interface UsernameCorrelationSectionProps {
  data: UsernameStats;
}

export default function UsernameCorrelationSection({ data }: UsernameCorrelationSectionProps) {
  const hasData = data.equals_username.count > 0 || data.contains_username.count > 0 || data.username_plus_suffix.count > 0;

  if (!hasData) {
    return null;
  }

  return (
    <Paper sx={{ p: 3, mb: 3 }}>
      <Typography variant="h5" gutterBottom>
        Username Correlation
      </Typography>
      <Typography variant="body2" color="text.secondary" sx={{ mb: 2 }}>
        Passwords that correlate with usernames
      </Typography>

      <TableContainer>
        <Table size="small">
          <TableHead>
            <TableRow>
              <TableCell sx={threeColumnTableStyles.labelCell}>Correlation Type</TableCell>
              <TableCell sx={threeColumnTableStyles.countCell}>Count</TableCell>
              <TableCell sx={threeColumnTableStyles.percentageCell}>Percentage</TableCell>
            </TableRow>
          </TableHead>
          <TableBody>
            {data.equals_username.count > 0 && (
              <TableRow>
                <TableCell sx={threeColumnTableStyles.labelCell}>Password Same as Username</TableCell>
                <TableCell sx={threeColumnTableStyles.countCell}>{data.equals_username.count.toLocaleString()}</TableCell>
                <TableCell sx={threeColumnTableStyles.percentageCell}>{data.equals_username.percentage.toFixed(2)}%</TableCell>
              </TableRow>
            )}
            {data.contains_username.count > 0 && (
              <TableRow>
                <TableCell sx={threeColumnTableStyles.labelCell}>Password Contains Username</TableCell>
                <TableCell sx={threeColumnTableStyles.countCell}>{data.contains_username.count.toLocaleString()}</TableCell>
                <TableCell sx={threeColumnTableStyles.percentageCell}>{data.contains_username.percentage.toFixed(2)}%</TableCell>
              </TableRow>
            )}
            {data.username_plus_suffix.count > 0 && (
              <TableRow>
                <TableCell sx={threeColumnTableStyles.labelCell}>Username Part of Password</TableCell>
                <TableCell sx={threeColumnTableStyles.countCell}>{data.username_plus_suffix.count.toLocaleString()}</TableCell>
                <TableCell sx={threeColumnTableStyles.percentageCell}>{data.username_plus_suffix.percentage.toFixed(2)}%</TableCell>
              </TableRow>
            )}
          </TableBody>
        </Table>
      </TableContainer>
    </Paper>
  );
}
