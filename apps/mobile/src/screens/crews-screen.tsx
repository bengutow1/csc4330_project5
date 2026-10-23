import {
  type Crew,
  errorMessage,
  fieldErrors,
  useCreateCrew,
  useJoinCrew,
  useLeaveCrew,
  useMyCrews,
} from "@fries-in-the-bag/api";
import { Stack, useRouter } from "expo-router";
import { useRef, useState } from "react";
import { KeyboardAvoidingView, ScrollView, StyleSheet, Text, TextInput, View } from "react-native";
import { useSafeAreaInsets } from "react-native-safe-area-context";
import { Button } from "@/components/button";
import { Card, Divider } from "@/components/card";
import { EmptyState } from "@/components/empty-state";
import { ErrorState } from "@/components/error-state";
import { FormSection } from "@/components/form-section";
import { Header } from "@/components/header";
import { Skeleton } from "@/components/skeleton";
import { memberLine } from "@/lib/crews";
import { COLORS, RADII, SPACE, TYPE } from "@/theme";

const ROW_SKELETON = 56;
const SKELETON_ROWS = ["first", "second"] as const;

/**
 * CrewsScreen is B3: the player's crews, each with its code to share and a way out, then making a
 * crew and joining one with a code. A crew's board is on the Board tab, under its name.
 */
export function CrewsScreen() {
  const router = useRouter();
  const insets = useSafeAreaInsets();
  const crews = useMyCrews();
  const create = useCreateCrew();
  const join = useJoinCrew();
  const leave = useLeaveCrew();
  const nameInput = useRef<TextInput>(null);
  const [name, setName] = useState("");
  const [code, setCode] = useState("");

  const make = () => create.mutate({ name }, { onSuccess: () => setName("") });
  const joinWith = () => join.mutate({ code }, { onSuccess: () => setCode("") });

  // A field's own message goes under it; any other refusal ("no crew has that code") goes there
  // too, since each form has one input.
  const nameError =
    fieldErrors(create.error).name ?? (create.isError ? errorMessage(create.error) : undefined);
  const codeError =
    fieldErrors(join.error).code ?? (join.isError ? errorMessage(join.error) : undefined);

  let list = null;
  if (crews.isPending) {
    list = (
      <View accessible accessibilityLabel="Loading your crews" style={styles.skeleton}>
        {SKELETON_ROWS.map((row) => (
          <Skeleton key={row} height={ROW_SKELETON} radius={RADII.md} />
        ))}
      </View>
    );
  } else if (crews.isError) {
    list = (
      <ErrorState
        title="Couldn't load your crews"
        error={crews.error}
        retrying={crews.isFetching}
        onRetry={() => void crews.refetch()}
      />
    );
  } else if (crews.data.crews.length === 0) {
    list = (
      <EmptyState
        title="No crews yet"
        line="Make one for your friends, or join theirs with a code."
        // Its own words: "Make a crew" is the form's button below, which a tap by label must find.
        action={{ label: "Name your crew", onPress: () => nameInput.current?.focus() }}
      />
    );
  } else {
    const last = crews.data.crews.length - 1;
    list = (
      <Card>
        {crews.data.crews.map((crew, i) => (
          <View key={String(crew.crewId)}>
            <CrewRow
              crew={crew}
              leaving={leave.isPending && leave.variables?.crewId === crew.crewId}
              onLeave={() => leave.mutate({ crewId: crew.crewId })}
            />
            {i === last ? null : <Divider />}
          </View>
        ))}
      </Card>
    );
  }

  return (
    <KeyboardAvoidingView
      behavior={process.env.EXPO_OS === "ios" ? "padding" : undefined}
      style={styles.root}
    >
      <Stack.Screen options={{ headerShown: false }} />
      <ScrollView
        contentContainerStyle={[styles.content, { paddingTop: insets.top }]}
        keyboardShouldPersistTaps="handled"
      >
        <Header
          title="Crews"
          subtitle="Private boards for you and your friends."
          onBack={() => router.back()}
        />

        <FormSection title="Your crews">
          {list}
          {leave.isError ? <Text style={styles.error}>{errorMessage(leave.error)}</Text> : null}
        </FormSection>

        <FormSection title="New crew" note="You get a code to share with your friends.">
          <TextInput
            ref={nameInput}
            accessibilityLabel="Crew name"
            maxLength={30}
            onChangeText={setName}
            onSubmitEditing={make}
            placeholder="Dorm 4B"
            placeholderTextColor={COLORS.muted}
            returnKeyType="done"
            style={styles.input}
            value={name}
          />
          {nameError ? <Text style={styles.error}>{nameError}</Text> : null}
          <Button
            label="Make a crew"
            pendingLabel="Making…"
            pending={create.isPending}
            onPress={make}
          />
        </FormSection>

        <FormSection title="Join a crew" note="Type the six-digit code a friend shared.">
          <TextInput
            accessibilityLabel="Crew code"
            inputMode="numeric"
            maxLength={7}
            onChangeText={setCode}
            onSubmitEditing={joinWith}
            // Words, not digits: a crew's real code is the only six digits on this screen.
            placeholder="Six-digit code"
            placeholderTextColor={COLORS.muted}
            returnKeyType="done"
            style={styles.input}
            value={code}
          />
          {codeError ? <Text style={styles.error}>{codeError}</Text> : null}
          <Button
            label="Join"
            variant="secondary"
            pendingLabel="Joining…"
            pending={join.isPending}
            onPress={joinWith}
          />
        </FormSection>
      </ScrollView>
    </KeyboardAvoidingView>
  );
}

/** CrewRow is one of the player's crews: its name, how many are in it, its code, and a way out. */
function CrewRow(props: { crew: Crew; leaving: boolean; onLeave: () => void }) {
  return (
    <View style={styles.row}>
      <View style={styles.words}>
        <Text style={styles.name}>{props.crew.name}</Text>
        <Text style={styles.line}>{memberLine(props.crew)}</Text>
      </View>
      {/* The code on its own, so it reads, and copies, as the one thing to share. */}
      <Text selectable style={styles.code}>
        {props.crew.code}
      </Text>
      <Button
        label="Leave"
        variant="secondary"
        size="small"
        pendingLabel="Leaving…"
        pending={props.leaving}
        onPress={props.onLeave}
      />
    </View>
  );
}

const styles = StyleSheet.create({
  root: { flex: 1, backgroundColor: COLORS.bg },
  content: { gap: SPACE.xl, paddingHorizontal: SPACE.lg, paddingBottom: SPACE.xxl },
  skeleton: { gap: SPACE.sm },
  row: { flexDirection: "row", alignItems: "center", gap: SPACE.md, paddingVertical: SPACE.sm },
  words: { flex: 1, gap: SPACE.xxs },
  name: { ...TYPE.bodyStrong, color: COLORS.ink },
  line: { ...TYPE.caption, color: COLORS.muted },
  code: { ...TYPE.bodyStrong, color: COLORS.ketchup, letterSpacing: 1 },
  input: {
    ...TYPE.body,
    height: 52,
    paddingHorizontal: SPACE.lg,
    borderWidth: 1,
    borderColor: COLORS.lineStrong,
    borderRadius: RADII.lg,
    backgroundColor: COLORS.surface,
    color: COLORS.ink,
  },
  error: { ...TYPE.small, color: COLORS.ketchupInk },
});
