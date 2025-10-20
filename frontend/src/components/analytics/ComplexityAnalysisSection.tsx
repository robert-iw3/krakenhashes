/**
 * Complexity analysis section showing all 16 character type categories.
 * Includes single type, two types, three types, four types, and complex short/long.
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
  Divider,
} from '@mui/material';
import { ComplexityStats } from '../../types/analytics';
import { threeColumnTableStyles } from './tableStyles';

interface ComplexityAnalysisSectionProps {
  data: ComplexityStats;
}

export default function ComplexityAnalysisSection({ data }: ComplexityAnalysisSectionProps) {
  // Helper to filter out zero counts
  const filterNonZero = (obj: Record<string, { count: number; percentage: number }>) => {
    return Object.entries(obj).filter(([_, value]) => value.count > 0);
  };

  const singleType = useMemo(() => filterNonZero(data.single_type), [data.single_type]);
  const twoTypes = useMemo(() => filterNonZero(data.two_types), [data.two_types]);
  const threeTypes = useMemo(() => filterNonZero(data.three_types), [data.three_types]);
  const fourTypes = data.four_types.count > 0 ? data.four_types : null;
  const complexShort = data.complex_short.count > 0 ? data.complex_short : null;
  const complexLong = data.complex_long.count > 0 ? data.complex_long : null;

  // Check if there's any data to display
  const hasData = singleType.length > 0 || twoTypes.length > 0 || threeTypes.length > 0 ||
                  fourTypes || complexShort || complexLong;

  if (!hasData) {
    return null;
  }

  const renderCategory = (label: string, stats: { count: number; percentage: number }) => (
    <TableRow>
      <TableCell sx={threeColumnTableStyles.labelCell}>{label}</TableCell>
      <TableCell sx={threeColumnTableStyles.countCell}>{stats.count.toLocaleString()}</TableCell>
      <TableCell sx={threeColumnTableStyles.percentageCell}>{stats.percentage.toFixed(2)}%</TableCell>
    </TableRow>
  );

  return (
    <Paper sx={{ p: 3, mb: 3 }}>
      <Typography variant="h5" gutterBottom>
        Character Complexity Analysis
      </Typography>
      <Typography variant="body2" color="text.secondary" sx={{ mb: 2 }}>
        Distribution of passwords by character type combinations
      </Typography>

      <TableContainer>
        <Table size="small">
          <TableHead>
            <TableRow>
              <TableCell sx={threeColumnTableStyles.labelCell}>Category</TableCell>
              <TableCell sx={threeColumnTableStyles.countCell}>Count</TableCell>
              <TableCell sx={threeColumnTableStyles.percentageCell}>Percentage</TableCell>
            </TableRow>
          </TableHead>
          <TableBody>
            {/* Single Type */}
            {singleType.length > 0 && (
              <>
                <TableRow sx={{ backgroundColor: 'action.hover' }}>
                  <TableCell colSpan={3}>
                    <strong>Single Character Type</strong>
                  </TableCell>
                </TableRow>
                {singleType.map(([name, stats]) => renderCategory(name, stats))}
              </>
            )}

            {/* Two Types */}
            {twoTypes.length > 0 && (
              <>
                <TableRow sx={{ backgroundColor: 'action.hover' }}>
                  <TableCell colSpan={3}>
                    <strong>Two Character Types</strong>
                  </TableCell>
                </TableRow>
                {twoTypes.map(([name, stats]) => renderCategory(name, stats))}
              </>
            )}

            {/* Three Types */}
            {threeTypes.length > 0 && (
              <>
                <TableRow sx={{ backgroundColor: 'action.hover' }}>
                  <TableCell colSpan={3}>
                    <strong>Three Character Types</strong>
                  </TableCell>
                </TableRow>
                {threeTypes.map(([name, stats]) => renderCategory(name, stats))}
              </>
            )}

            {/* Four Types */}
            {fourTypes && (
              <>
                <TableRow sx={{ backgroundColor: 'action.hover' }}>
                  <TableCell colSpan={3}>
                    <strong>Four Character Types</strong>
                  </TableCell>
                </TableRow>
                {renderCategory('All Character Types', fourTypes)}
              </>
            )}

            {/* Complex Short/Long */}
            {(complexShort || complexLong) && (
              <>
                <TableRow sx={{ backgroundColor: 'action.hover' }}>
                  <TableCell colSpan={3}>
                    <strong>Complex Passwords (3-4 types)</strong>
                  </TableCell>
                </TableRow>
                {complexShort && renderCategory('Complex (≤14 characters)', complexShort)}
                {complexLong && renderCategory('Complex (15+ characters)', complexLong)}
              </>
            )}
          </TableBody>
        </Table>
      </TableContainer>
    </Paper>
  );
}
